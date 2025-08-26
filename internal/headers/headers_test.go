package headers

import (
   "testing"

   "github.com/stretchr/testify/assert"
   "github.com/stretchr/testify/require"

)

func TestHeaderParse(t *testing.T) {

	// Test: Valid single header
	headers := NewHeaders()
	data := []byte("Host: localhost:42069\r\nFooFoo:      barbar      \r\n\r\n")
	n, done, err := headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	host, ok := headers.Get("HOST")
	assert.True(t, ok)
	assert.Equal(t, "localhost:42069", host)

	foofoo, ok := headers.Get("FOOFOO")
	assert.True(t, ok)
	assert.Equal(t, "barbar", foofoo)

	missingKey, ok := headers.Get("MISSINGKEY")
	assert.False(t, ok)
	assert.Equal(t, "", missingKey)
	assert.Equal(t, 52, n)
	assert.True(t, done)

	// Test: Invalid spacing header
	headers = NewHeaders()
	//data = []byte("       Host : localhost:42069       \r\n\r\n")
	data = []byte("H©st: localhost:42069\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)


	// Test: Check for addtional field-name values
	headers = NewHeaders()
	data = []byte("Host: localhost:42069\r\nHost: localhost:1702\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)

	host, ok = headers.Get("HOST")
	assert.True(t, ok)
	assert.Equal(t, "localhost:42069,localhost:1702", host)
	assert.False(t, done)

}
