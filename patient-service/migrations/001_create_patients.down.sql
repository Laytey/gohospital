-- Откат миграции 001: удаляем таблицу пациентов
DROP INDEX IF EXISTS idx_patients_name;
DROP TABLE IF EXISTS patients;