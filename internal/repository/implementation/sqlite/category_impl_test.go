package sqlite_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// CategoryRepositorySuite is intentionally stubbed during Feature 109 Loop 1.
// Loop 2 rewrites these tests against the new name-keyed schema and the
// reshaped repository.CategoryRepository interface.
type CategoryRepositorySuite struct {
	suite.Suite
}

func TestCategory(t *testing.T) {
	suite.Run(t, new(CategoryRepositorySuite))
}

func (s *CategoryRepositorySuite) TestPlaceholder() {
	s.T().Skip("rewritten in Feature 109 Loop 2")
}
