package errors

import "fmt"

var russianMessages = map[ErrorCode]string{
	ErrValidation:    "Ошибка валидации данных",
	ErrNotFound:      "Ресурс не найден",
	ErrStorage:       "Ошибка хранения данных",
	ErrCalculation:   "Ошибка расчета",
	ErrConfiguration: "Ошибка конфигурации",
	ErrSecurity:      "Ошибка безопасности",
	ErrNotification:  "Ошибка уведомления",
	ErrBusinessLogic: "Ошибка бизнес-логики",
}

func GetUserFriendlyMessage(err error) string {
	var appErr *AppError
	if e, ok := err.(*AppError); ok {
		appErr = e
	} else {
		return "Произошла непредвиденная ошибка"
	}

	baseMessage, exists := russianMessages[appErr.Code]
	if !exists {
		baseMessage = "Произошла ошибка"
	}

	switch appErr.Code {
	case ErrValidation:
		if details, ok := appErr.Details["field"]; ok {
			return fmt.Sprintf("%s: поле %s", baseMessage, details)
		}
	case ErrNotFound:
		if resource, ok := appErr.Details["resource"]; ok {
			if id, ok := appErr.Details["id"]; ok {
				return fmt.Sprintf("%s: %s с идентификатором '%s' не найден", baseMessage, resource, id)
			}
			return fmt.Sprintf("%s: %s не найден", baseMessage, resource)
		}
	case ErrStorage:
		if operation, ok := appErr.Details["operation"]; ok {
			return fmt.Sprintf("%s при выполнении операции: %s", baseMessage, operation)
		}
	}

	return baseMessage
}

func GetDetailedUserMessage(err error) string {
	userMsg := GetUserFriendlyMessage(err)

	if appErr, ok := err.(*AppError); ok && appErr.Original != nil {
		return fmt.Sprintf("%s (%v)", userMsg, appErr.Original)
	}

	return userMsg
}
