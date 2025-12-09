package domain

import "errors"

var ErrInvalidOperation = errors.New("invalid operation")

type Operation struct {
	value string
}

var (
	OperationDownload = Operation{value: "download"}
	OperationUpload   = Operation{value: "upload"}
)

func ParseOperation(s string) (Operation, error) {
	switch s {
	case OperationDownload.value:
		return OperationDownload, nil
	case OperationUpload.value:
		return OperationUpload, nil
	default:
		return Operation{}, ErrInvalidOperation
	}
}

func (o Operation) String() string {
	return o.value
}

func (o Operation) IsZero() bool {
	return o.value == ""
}
