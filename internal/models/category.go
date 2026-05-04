package models

import "time"

type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var DefaultCategories = []Category{
	{ID: "salary", Name: "Зарплата"},
	{ID: "deposit_interest", Name: "Проценты по вкладам"},
	{ID: "food", Name: "Еда"},
	{ID: "transport", Name: "Транспорт"},
	{ID: "subscriptions", Name: "Подписки"},
	{ID: "housing", Name: "Жилье"},
	{ID: "health", Name: "Здоровье"},
	{ID: "education", Name: "Обучение"},
	{ID: "investments", Name: "Инвестиции"},
	{ID: "emergency_fund", Name: "Финансовая подушка"},
	{ID: "entertainment", Name: "Развлечения"},
	{ID: "other", Name: "Прочее"},
}
