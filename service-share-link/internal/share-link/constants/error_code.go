package constants

import "errors"

var ErrorRecordDeleteNotChange = errors.New("Record delete not change")
var ErrorRecordNotFound = errors.New("Record not found")
var ErrorSever = errors.New("Service internal")
var ErrQueryConflict = errors.New("[Query] Insert conflict code")
var ErrQueryResourceConflict = errors.New("[Query] Insert conflict live link for resource")

var ErrLinkDeleted = errors.New("Link is deleted by its owner")
var ErrLinkDisable = errors.New("Link is disabled by its owner")
var ErrNotOwner = errors.New("Link is not owned by requester")
var ErrResourceSharedByOther = errors.New("Resource is already shared by another user")
