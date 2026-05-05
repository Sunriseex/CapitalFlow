package models

import "time"

type Category struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var DefaultCategories = []Category{
	{Slug: "salary", Name: "Зарплата"},
	{Slug: "deposit_interest", Name: "Проценты по вкладам"},
	{Slug: "food", Name: "Еда"},
	{Slug: "transport", Name: "Транспорт"},
	{Slug: "subscriptions", Name: "Подписки"},
	{Slug: "housing", Name: "Жилье"},
	{Slug: "health", Name: "Здоровье"},
	{Slug: "education", Name: "Обучение"},
	{Slug: "investments", Name: "Инвестиции"},
	{Slug: "emergency_fund", Name: "Финансовая подушка"},
	{Slug: "entertainment", Name: "Развлечения"},
	{Slug: "other", Name: "Прочее"},
}
