package api

import (
	log "github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func MissingRequiredFieldError(field, fieldDescription string) *status.Status {
	stat := status.New(codes.InvalidArgument, "missing model field")

	fieldViolations := make([]*errdetails.BadRequest_FieldViolation, 0)
	fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
		Field:       field,
		Description: fieldDescription,
	})

	badrequest := &errdetails.BadRequest{
		FieldViolations: fieldViolations,
	}

	statWithDetails, err := stat.WithDetails(badrequest)
	if err != nil {
		log.Error("unexpected error, unable to build status object: ", field, fieldDescription)
		return stat
	}

	return statWithDetails
}
