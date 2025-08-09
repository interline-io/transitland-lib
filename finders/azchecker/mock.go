package azchecker

import (
	"context"
	"errors"
	"strings"

	"github.com/interline-io/transitland-mw/auth/authn"
)

type MockUserProvider struct {
	users map[string]authn.User
}

func NewMockUserProvider() *MockUserProvider {
	return &MockUserProvider{
		users: map[string]authn.User{},
	}
}

func (c *MockUserProvider) AddUser(key string, u authn.User) {
	c.users[key] = authn.NewCtxUser(u.ID(), u.Name(), u.Email())
}

func (c *MockUserProvider) UserByID(ctx context.Context, id string) (authn.User, error) {
	if user, ok := c.users[id]; ok {
		return user, nil
	}
	return nil, errors.New("unauthorized")
}

func (c *MockUserProvider) Users(ctx context.Context, userQuery string) ([]authn.User, error) {
	var ret []authn.User
	uq := strings.ToLower(userQuery)
	for _, user := range c.users {
		user := user
		un := strings.ToLower(user.Name())
		if userQuery == "" || strings.Contains(un, uq) {
			ret = append(ret, user)
		}
	}
	return ret, nil
}

//////

type MockFGAClient struct{}

func NewMockFGAClient() *MockFGAClient {
	return &MockFGAClient{}
}

func (c *MockFGAClient) Check(context.Context, TupleKey, ...TupleKey) (bool, error) {
	return false, nil
}

func (c *MockFGAClient) ListObjects(context.Context, TupleKey) ([]TupleKey, error) {
	return nil, nil
}

func (c *MockFGAClient) GetObjectTuples(context.Context, TupleKey) ([]TupleKey, error) {
	return nil, nil
}

func (c *MockFGAClient) WriteTuple(context.Context, TupleKey) error {
	return nil
}

func (c *MockFGAClient) SetExclusiveRelation(context.Context, TupleKey) error {
	return nil
}

func (c *MockFGAClient) SetExclusiveSubjectRelation(context.Context, TupleKey, ...Relation) error {
	return nil
}

func (c *MockFGAClient) DeleteTuple(context.Context, TupleKey) error {
	return nil
}
