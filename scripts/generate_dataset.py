#!/usr/bin/env python3
"""Deterministic 1C-style CSV dataset generator (1000+ records)."""
import csv
import random
import datetime
import os

SEED = 42
TOTAL_RECORDS = 1000

random.seed(SEED)

BRANDS = [
    "Toyota", "Hyundai", "Kia", "Volkswagen", "Renault",
    "Lada", "BMW", "Mercedes-Benz", "Audi", "Lexus",
    "Mazda", "Nissan", "Honda", "Chevrolet", "Ford",
    "Skoda", "Peugeot", "Mitsubishi", "Subaru", "Volvo",
]

MODELS = {
    "Toyota": ["Camry", "Corolla", "RAV4", "Land Cruiser", "Highlander", "Hilux"],
    "Hyundai": ["Tucson", "Creta", "Santa Fe", "Elantra", "Sonata", "Palisade"],
    "Kia": ["Sportage", "Rio", "Seltos", "Sorento", "K5", "Carnival"],
    "Volkswagen": ["Polo", "Tiguan", "Passat", "Touareg", "Golf", "Jetta"],
    "Renault": ["Duster", "Kaptur", "Arkana", "Megane", "Logan", "Sandero"],
    "Lada": ["Granta", "Vesta", "XRAY", "Niva", "Largus", "Priora"],
    "BMW": ["X5", "X3", "5 Series", "3 Series", "X1", "7 Series"],
    "Mercedes-Benz": ["E-Class", "C-Class", "GLE", "GLC", "S-Class", "A-Class"],
    "Audi": ["Q5", "A4", "Q7", "A6", "Q3", "A8"],
    "Lexus": ["RX", "LX", "ES", "NX", "GX", "LS"],
    "Mazda": ["CX-5", "CX-9", "Mazda6", "CX-30", "CX-4", "MX-5"],
    "Nissan": ["Qashqai", "X-Trail", "Patrol", "Navara", "Juke", "Murano"],
    "Honda": ["CR-V", "Civic", "Accord", "HR-V", "Pilot", "Fit"],
    "Chevrolet": ["Malibu", "Equinox", "Tahoe", "Traverse", "Camaro", "Spark"],
    "Ford": ["Focus", "Explorer", "Mustang", "Escape", "F-150", "Edge"],
    "Skoda": ["Octavia", "Kodiaq", "Karoq", "Superb", "Fabia", "Scala"],
    "Peugeot": ["3008", "2008", "5008", "508", "308", "Partner"],
    "Mitsubishi": ["Outlander", "Pajero", "ASX", "Lancer", "Eclipse Cross", "Pajero Sport"],
    "Subaru": ["Forester", "Outback", "XV", "Legacy", "WRX", "BRZ"],
    "Volvo": ["XC60", "XC90", "S60", "V60", "XC40", "S90"],
}

COLORS = [
    "Белый", "Черный", "Серебристый", "Красный", "Синий",
    "Серый", "Зеленый", "Коричневый", "Оранжевый", "Желтый",
    "Бежевый", "Фиолетовый", "Голубой", "Бордовый", "Хаки",
]

TRANSMISSIONS = ["AT", "MT", "CVT", "DCT", "AMT"]
BODY_TYPES = ["Седан", "Хэтчбек", "SUV", "Кроссовер", "Универсал", "Пикап", "Минивэн", "Купе", "Кабриолет", "Лифтбэк"]
ENGINES = [
    "2.0 бензин 150 л.с.", "1.6 бензин 110 л.с.", "2.5 бензин 186 л.с.",
    "3.5 бензин 249 л.с.", "1.8 гибрид 122 л.с.", "2.0 дизель 150 л.с.",
    "1.4 турбо 140 л.с.", "2.0 турбо 240 л.с.", "3.0 дизель 245 л.с.",
    "1.6 дизель 116 л.с.", "2.4 бензин 173 л.с.", "5.0 бензин 422 л.с.",
]
DEFECTS = [
    "", "", "", "", "", "", "", "", "", "",  # mostly no defects
    "Царапина на бампере", "Проблемы с АКПП", "Двигатель троит",
    "Требуется замена масла", "Стук в подвеске", "Не работает кондиционер",
    "Повреждение лакокрасочного покрытия", "Износ тормозных колодок",
    "Утечка антифриза", "Шум в коробке передач", "Проблемы с электроникой",
    "Требуется замена ремня ГРМ", "Подвески требуют обслуживания",
]
STATUSES = ["active", "sold", "reserved"]

VIN_CHARS = "ABCDEFGHJKLMNPRSTUVWXYZ0123456789"


def generate_vin(rng):
    return "".join(rng.choice(VIN_CHARS) for _ in range(17))


def random_date(rng):
    days_ago = rng.randint(0, 730)
    d = datetime.date.today() - datetime.timedelta(days=days_ago)
    return d.strftime("%Y-%m-%d")


def generate_problematic_row(problem_type, rng):
    """Generate intentionally problematic rows for validation testing."""
    base = [
        "TOYOTA", "Camry", "2020", "50000", "5000000",
        "KZT", "Белый", "2.0 бензин 150 л.с.", "AT", "Седан",
        "", "active", "2024-01-01",
    ]
    if problem_type == "missing_vin":
        base[0] = ""
    elif problem_type == "invalid_mileage":
        base[3] = "abc"
    elif problem_type == "invalid_year":
        base[2] = "1800"
    elif problem_type == "missing_brand":
        base[1] = ""
    elif problem_type == "missing_model":
        base[2] = ""
        base[1] = "Corolla"
    elif problem_type == "malformed_columns":
        return ["SHORT", "Row"]  # too few columns
    return base


def main():
    rng = random.Random(SEED)
    records = []

    # Header
    records.append([
        "VIN", "Brand", "Model", "Year", "MileageKm", "Price",
        "Currency", "Color", "Engine", "Transmission", "BodyType",
        "DefectsRaw", "Status", "UpdatedAt",
    ])

    problematic_rows = {
        5: "missing_vin",
        50: "invalid_mileage",
        100: "invalid_year",
        200: "missing_brand",
        500: "malformed_columns",
        750: "missing_model",
    }

    for i in range(TOTAL_RECORDS):
        row_num = i + 2  # 1-based with header

        if row_num in problematic_rows:
            records.append(generate_problematic_row(problematic_rows[row_num], rng))
            continue

        brand = rng.choice(BRANDS)
        model = rng.choice(MODELS[brand])
        year = 2000 + rng.randint(0, 25)
        mileage = rng.randint(0, 200000)
        price = 500000 + rng.randint(0, 15000000)
        color = rng.choice(COLORS)
        transmission = rng.choice(TRANSMISSIONS)
        body_type = rng.choice(BODY_TYPES)
        engine = rng.choice(ENGINES)
        defects = rng.choice(DEFECTS)
        status = rng.choice(STATUSES)
        updated_at = random_date(rng)
        vin = generate_vin(rng)

        records.append([
            vin, brand, model, str(year), str(mileage),
            str(price), "KZT", color, engine, transmission,
            body_type, defects, status, updated_at,
        ])

    output_path = os.path.join("data", "imports", "partner_1c_export.csv")
    os.makedirs(os.path.dirname(output_path), exist_ok=True)

    with open(output_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f, delimiter=";", quoting=csv.QUOTE_MINIMAL, lineterminator="\r\n")
        writer.writerows(records)

    print(f"Generated {len(records) - 1} data rows + 1 header to {output_path}")


if __name__ == "__main__":
    main()
