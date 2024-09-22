package error

import (
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AdmitError struct {
	message string
	code    int32
	reason  metav1.StatusReason
	causes  []metav1.StatusCause
}

func (e AdmitError) Error() string {
	return e.message
}

func (e AdmitError) AsResult() *metav1.Status {
	status := metav1.Status{
		Status:  "Failure",
		Message: e.message,
		Code:    e.code,
		Reason:  e.reason,
	}

	if len(e.causes) > 0 {
		status.Details = &metav1.StatusDetails{
			Causes: e.causes,
		}
	}

	return &status
}

// BadRequest return bad request error with code 400
func BadRequest(message string) AdmitError {
	return AdmitError{
		code:    http.StatusBadRequest,
		message: message,
		reason:  metav1.StatusReasonBadRequest,
	}
}

// MethodNotAllowed return method not allowed with code 405
func MethodNotAllowed(message string) AdmitError {
	return AdmitError{
		code:    http.StatusMethodNotAllowed,
		message: message,
		reason:  metav1.StatusReasonMethodNotAllowed,
	}
}

// InvalidError with error message with code 422
func InvalidError(message string, field string) AdmitError {
	return AdmitError{
		code:    http.StatusUnprocessableEntity,
		message: message,
		reason:  metav1.StatusReasonInvalid,
		causes: []metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: message,
				Field:   field,
			},
		},
	}
}

// StatusConflict return error message with code 409
func StatusConflict(message string) AdmitError {
	return AdmitError{
		code:    http.StatusConflict,
		message: message,
		reason:  metav1.StatusReasonConflict,
	}
}

// InternalError return error message with code 500
func InternalError(message string) AdmitError {
	return AdmitError{
		code:    http.StatusInternalServerError,
		message: message,
		reason:  metav1.StatusReasonInternalError,
	}
}
