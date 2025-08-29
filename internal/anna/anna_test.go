package anna

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	query := "Children in Soviet Russia by Deana Levin"
	expectedMD5 := "1aad53ca237e632662eff7f18fc4ba81"

	books, err := Search(query)
	assert.NoError(t, err)
	assert.Len(t, books, 1)
	assert.Equal(t, expectedMD5, books[0].Hash)
}
