package handlers

type validationError string

func (e validationError) Error() string {
	return string(e)
}

func errValidation(message string) error {
	return validationError(message)
}
