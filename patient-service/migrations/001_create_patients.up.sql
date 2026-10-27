-- Таблица пациентов
CREATE TABLE IF NOT EXISTS patients (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    birthdate DATE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Индекс для поиска по имени
CREATE INDEX IF NOT EXISTS idx_patients_name ON patients(name);