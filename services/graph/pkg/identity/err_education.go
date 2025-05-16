package identity

import (
	"context"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

// ErrEducationBackend is a dummy EducationBackend, doing nothing
type ErrEducationBackend struct{}

// CreateEducationSchool creates the supplied school in the identity backend.
func (i *ErrEducationBackend) CreateEducationSchool(ctx context.Context, school libregraph.EducationSchool) (*libregraph.EducationSchool, error) {
	return nil, errNotImplemented
}

// DeleteEducationSchool deletes a given school, identified by id
func (i *ErrEducationBackend) DeleteEducationSchool(ctx context.Context, id string) error {
	return errNotImplemented
}

// GetEducationSchool implements the EducationBackend interface for the ErrEducationBackend backend.
func (i *ErrEducationBackend) GetEducationSchool(ctx context.Context, nameOrID string) (*libregraph.EducationSchool, error) {
	return nil, errNotImplemented
}

// GetEducationSchools implements the EducationBackend interface for the ErrEducationBackend backend.
func (i *ErrEducationBackend) GetEducationSchools(ctx context.Context) ([]*libregraph.EducationSchool, error) {
	return nil, errNotImplemented
}

// UpdateEducationSchool implements the EducationBackend interface for the ErrEducationBackend backend.
func (i *ErrEducationBackend) UpdateEducationSchool(ctx context.Context, numberOrID string, school libregraph.EducationSchool) (*libregraph.EducationSchool, error) {
	return nil, errNotImplemented
}

// GetEducationSchoolUsers implements the EducationBackend interface for the ErrEducationBackend backend.
func (i *ErrEducationBackend) GetEducationSchoolUsers(ctx context.Context, id string) ([]*libregraph.EducationUser, error) {
	return nil, errNotImplemented
}

// GetEducationSchoolClasses implements the EducationBackend interface for the ErrEducationBackend backend.
func (i *ErrEducationBackend) GetEducationSchoolClasses(ctx context.Context, schoolNumberOrID string) ([]*libregraph.EducationClass, error) {
	return nil, errNotImplemented
}

// AddClassesToEducationSchool implements the EducationBackend interface for the ErrEducationBackend backend.
func (i *ErrEducationBackend) AddClassesToEducationSchool(ctx context.Context, schoolNumberOrID string, memberIDs []string) error {
	return errNotImplemented
}

// RemoveClassFromEducationSchool implements the EducationBackend interface for the ErrEducationBackend backend.
func (i *ErrEducationBackend) RemoveClassFromEducationSchool(ctx context.Context, schoolNumberOrID string, memberID string) error {
	return errNotImplemented
}

// AddUsersToEducationSchool adds new members (reference by a slice of IDs) to supplied school in the identity backend.
func (i *ErrEducationBackend) AddUsersToEducationSchool(ctx context.Context, schoolID string, memberID []string) error {
	return errNotImplemented
}

// RemoveUserFromEducationSchool removes a single member (by ID) from a school
func (i *ErrEducationBackend) RemoveUserFromEducationSchool(ctx context.Context, schoolID string, memberID string) error {
	return errNotImplemented
}

// GetEducationClasses implements the EducationBackend interface
func (i *ErrEducationBackend) GetEducationClasses(ctx context.Context) ([]*libregraph.EducationClass, error) {
	return nil, errNotImplemented
}

// GetEducationClass implements the EducationBackend interface
func (i *ErrEducationBackend) GetEducationClass(ctx context.Context, namedOrID string) (*libregraph.EducationClass, error) {
	return nil, errNotImplemented
}

// CreateEducationClass implements the EducationBackend interface
func (i *ErrEducationBackend) CreateEducationClass(ctx context.Context, class libregraph.EducationClass) (*libregraph.EducationClass, error) {
	return nil, errNotImplemented
}

// DeleteEducationClass implements the EducationBackend interface
func (i *ErrEducationBackend) DeleteEducationClass(ctx context.Context, nameOrID string) error {
	return errNotImplemented
}

// GetEducationClassMembers implements the EducationBackend interface
func (i *ErrEducationBackend) GetEducationClassMembers(ctx context.Context, nameOrID string) ([]*libregraph.EducationUser, error) {
	return nil, errNotImplemented
}

// UpdateEducationClass implements the EducationBackend interface
func (i *ErrEducationBackend) UpdateEducationClass(ctx context.Context, id string, class libregraph.EducationClass) (*libregraph.EducationClass, error) {
	return nil, errNotImplemented
}

// CreateEducationUser creates a given education user in the identity backend.
func (i *ErrEducationBackend) CreateEducationUser(ctx context.Context, user libregraph.EducationUser) (*libregraph.EducationUser, error) {
	return nil, errNotImplemented
}

// DeleteEducationUser deletes a given education user, identified by username or id, from the backend
func (i *ErrEducationBackend) DeleteEducationUser(ctx context.Context, nameOrID string) error {
	return errNotImplemented
}

// UpdateEducationUser applies changes to given education user, identified by username or id
func (i *ErrEducationBackend) UpdateEducationUser(ctx context.Context, nameOrID string, user libregraph.EducationUser) (*libregraph.EducationUser, error) {
	return nil, errNotImplemented
}

// GetEducationUser implements the EducationBackend interface for the ErrEducationBackend backend.
func (i *ErrEducationBackend) GetEducationUser(ctx context.Context, nameOrID string) (*libregraph.EducationUser, error) {
	return nil, errNotImplemented
}

// GetEducationUsers implements the EducationBackend interface for the ErrEducationBackend backend.
func (i *ErrEducationBackend) GetEducationUsers(ctx context.Context) ([]*libregraph.EducationUser, error) {
	return nil, errNotImplemented
}

// GetEducationClassTeachers implements the EducationBackend interface for the ErrEducationBackend backend.
func (i *ErrEducationBackend) GetEducationClassTeachers(ctx context.Context, classID string) ([]*libregraph.EducationUser, error) {
	return nil, errNotImplemented
}

// AddTeacherToEducationClass implements the EducationBackend interface for the ErrEducationBackend backend.
func (i *ErrEducationBackend) AddTeacherToEducationClass(ctx context.Context, classID string, teacherID string) error {
	return errNotImplemented
}

// RemoveTeacherFromEducationClass implements the EducationBackend interface for the ErrEducationBackend backend.
func (i *ErrEducationBackend) RemoveTeacherFromEducationClass(ctx context.Context, classID string, teacherID string) error {
	return errNotImplemented
}
