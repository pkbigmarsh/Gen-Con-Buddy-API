package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCategory_RejectsPremierEvent(t *testing.T) {
	err := ValidateCategory("Premier Event")
	require.Error(t, err, "Premier Event is no longer a valid SpecialCategory")
}
