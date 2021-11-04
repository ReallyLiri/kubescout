package dedup

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_normalizeMessage(t *testing.T) {
	assert.Equal(t, "", NormalizeTemporal(""))
	assert.Equal(t, "abc", NormalizeTemporal("abc"))
	assert.Equal(t, "hello world", NormalizeTemporal("hello world"))
	assert.Equal(t, "", NormalizeTemporal("<t>hello world</t>"))
	assert.Equal(t, "", NormalizeTemporal("<t></t>"))
	assert.Equal(t, "The  is here", NormalizeTemporal("The <t>hello world</t> is here"))
	assert.Equal(t, "The  brown  jumps  the  dog", NormalizeTemporal("The <t>quick</t> brown <t>fox</t> jumps <t>over</t> the <t>lazy</t> dog"))
	assert.Equal(t, "t<t>t", NormalizeTemporal("t<t>t"))
	assert.Equal(t, "t</t>a<t>t", NormalizeTemporal("t</t>a<t>t"))
	assert.Equal(t, "tt", NormalizeTemporal("t<t>t<t></t>t"))
	assert.Equal(t, "tt", NormalizeTemporal("t<t>t<t/></t>t"))
	assert.Equal(t, "t</t>t", NormalizeTemporal("t<t>t</t></t>t"))
}

func Test_cleanMessage(t *testing.T) {
	assert.Equal(t, "", CleanTemporal(""))
	assert.Equal(t, "abc", CleanTemporal("abc"))
	assert.Equal(t, "hello world", CleanTemporal("hello world"))
	assert.Equal(t, "hello world", CleanTemporal("<t>hello world</t>"))
	assert.Equal(t, "", CleanTemporal("<t></t>"))
	assert.Equal(t, "The hello world is here", CleanTemporal("The <t>hello world</t> is here"))
	assert.Equal(t, "The quick brown fox jumps over the lazy dog", CleanTemporal("The <t>quick</t> brown <t>fox</t> jumps <t>over</t> the <t>lazy</t> dog"))
	assert.Equal(t, "tt", CleanTemporal("t<t>t"))
	assert.Equal(t, "tat", CleanTemporal("t</t>a<t>t"))
	assert.Equal(t, "ttt", CleanTemporal("t<t>t<t></t>t"))
	assert.Equal(t, "tt<t/>t", CleanTemporal("t<t>t<t/></t>t"))
	assert.Equal(t, "ttt", CleanTemporal("t<t>t</t></t>t"))
}
