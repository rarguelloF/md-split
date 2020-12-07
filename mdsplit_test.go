package mdsplit

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkdownSplit(t *testing.T) {
	t.Parallel()

	type testInput struct {
		markdown string
		max      int
		join     string
	}

	type testOutput struct {
		chunks []string
		ok     bool
	}

	testCases := []struct {
		input    *testInput
		expected *testOutput
	}{
		{
			&testInput{"Some basic comment", 100, ""},
			&testOutput{[]string{"Some basic comment"}, true},
		},
		{
			&testInput{"Some basic comment", 10, ""},
			&testOutput{[]string{"Some basic", " comment"}, true},
		},
		{
			&testInput{"### Comment with title\n\nIncludes the title in every split.", 55, ""},
			&testOutput{
				[]string{
					"### Comment with title (1/3)\n\nIncludes the ",
					"### Comment with title (2/3)\n\ntitle in ever",
					"### Comment with title (3/3)\n\ny split.",
				},
				true,
			},
		},
		{
			&testInput{"```thelang\nSplits codeblocks.\nProperly\nand without breaking syntax highlight```", 30, ""},
			&testOutput{
				[]string{
					"```thelang\nSplits codebloc\n```",
					"```thelang\nks.\nProperly\nan\n```",
					"```thelang\nd without break\n```",
					"```thelang\ning syntax high\n```",
					"```thelang\nlight\n```",
				},
				true,
			},
		},
		{
			&testInput{"<tag1>Splits content <tag2> nested in html spans <tag3>properly</tag3> and keeping tags.</tag2></tag1>", 60, ""},
			&testOutput{
				[]string{
					"<tag1>Splits content </tag1>",
					"<tag2><tag1> nested in html spans </tag1></tag2>",
					"<tag3><tag2><tag1>properly</tag1></tag2></tag3>",
					"<tag2><tag1> and keeping tags.</tag1></tag2>",
				},
				true,
			},
		},
		{
			&testInput{"<tag1>Splits content <tag2> nested in html spans <tag3>properly</tag3> and keeping tags.</tag2></tag1>", 40, ""},
			&testOutput{
				[]string{
					"<tag1>Splits content </tag1>",
					"<tag2><tag1> nested in htm</tag1></tag2>",
					"<tag2><tag1>l spans </tag1></tag2>",
					"<tag3><tag2><tag1>p</tag1></tag2></tag3>",
					"<tag3><tag2><tag1>r</tag1></tag2></tag3>",
					"<tag3><tag2><tag1>o</tag1></tag2></tag3>",
					"<tag3><tag2><tag1>p</tag1></tag2></tag3>",
					"<tag3><tag2><tag1>e</tag1></tag2></tag3>",
					"<tag3><tag2><tag1>r</tag1></tag2></tag3>",
					"<tag3><tag2><tag1>l</tag1></tag2></tag3>",
					"<tag3><tag2><tag1>y</tag1></tag2></tag3>",
					"<tag2><tag1> and keeping t</tag1></tag2>",
					"<tag2><tag1>ags.</tag1></tag2>",
				},
				true,
			},
		},
		{
			// if given max length is not big enough to fit all corresponding opening and closing spans, perform simple split
			&testInput{"<tag1>Splits content <tag2> nested in html spans <tag3>properly</tag3> and keeping tags.</tag2></tag1>", 30, ""},
			&testOutput{
				[]string{
					"<tag1>Splits content <tag2> ne",
					"sted in html spans <tag3>prope",
					"rly</tag3> and keeping tags.</",
					"tag2></tag1>",
				},
				false,
			},
		},
		{
			&testInput{"# Main title\n\nSome text.\n\n## Second title\n\nWhatever", 40, ""},
			&testOutput{
				[]string{
					"# Main title (1/7)\n\nSome tex",
					"# Main title (2/7)\n\nt.",
					"# Main title (3/7)\n\n## Sec\n\n",
					"# Main title (4/7)\n\n## ond\n\n",
					"# Main title (5/7)\n\n##  ti\n\n",
					"# Main title (6/7)\n\n## tle\n\n",
					"# Main title (7/7)\n\nWhatever",
				},
				true,
			},
		},

		// TODO: smart split of tables is not supported yet, update test when implemented
		{
			&testInput{
				markdown: `
| A     | B          | This one has a very long heading | D      | E       |
|-------|------------|----------------------------------|--------|---------|
| Text  | Text       | More text                        | Whaaat | Heyyy   |
| C     | asnmdnasnd | Foo                              | Pepito | owewoie |
| iiiii | oooo       | Bar                              | a      | lhgkgk  |
`,
				max:  100,
				join: "",
			},
			&testOutput{
				chunks: []string{
					"| A     | B          | This one has a very long heading | D      | E       |\n|-------|------------|-",
					"---------------------------------|--------|---------|\n| Text  | Text       | More text              ",
					"          | Whaaat | Heyyy   |\n| C     | asnmdnasnd | Foo                              | Pepito | ow",
					"ewoie |\n| iiiii | oooo       | Bar                              | a      | lhgkgk  |",
				},
				ok: true,
			},
		},
		{
			&testInput{"[I'm an inline-style link](https://www.google.com)", 40, ""},
			&testOutput{
				[]string{
					"[I'm an inline-](https://www.google.com)",
					"[style link](https://www.google.com)",
				},
				true,
			},
		},
		{
			&testInput{`
[I'm an inline-style link](https://www.google.com)

[I'm an inline-style link with title](https://www.google.com "Google's Homepage")

[I'm a reference-style link][Arbitrary case-insensitive reference text]

[I'm a relative reference to a repository file](../blob/master/LICENSE)

[You can use numbers for reference-style link definitions][1]

Or leave it empty and use the [link text itself].

URLs and URLs in angle brackets will automatically get turned into links. 
http://www.example.com or <http://www.example.com> and sometimes 
example.com (but not on Github, for example).

Some text to show that the reference links can follow later.

[arbitrary case-insensitive reference text]: https://www.mozilla.org
[1]: http://slashdot.org
[link text itself]: http://www.reddit.com
`, 100, ""},
			&testOutput{
				[]string{
					"[I'm an inline-style link](https://www.google.com)",
					"[I'm an inline-style link with title](https://www.google.com \"Google's Homepage\")",
					"[I'm a reference-style link](https://www.mozilla.org)",
					"[I'm a relative reference to a repository file](../blob/master/LICENSE)",
					"[You can use numbers for reference-style link definitions](http://slashdot.org)",
					"Or leave it empty and use the [link text itself](http://www.reddit.com).",
					"URLs and URLs in angle brackets will automatically get turned into links.\nhttp://www.example.com or ",
					"[http://www.example.com](http://www.example.com) and sometimes",
					"\nexample.com (but not on Github, for example).",
					"Some text to show that the reference links can follow later.",
				},
				true,
			},
		},
		{
			&testInput{`
1. First ordered list item
2. Another item
⋅⋅* Unordered sub-list. 
1. Actual numbers don't matter, just that it's a number
⋅⋅1. Ordered sub-list
4. And another item.
`, 40, ""},
			&testOutput{
				[]string{
					"\n1. First ordered list item\n2. Another i",
					"tem\n⋅⋅* Unordered sub-list. \n1. Actu",
					"al numbers don't matter, just that it's ",
					"a number\n⋅⋅1. Ordered sub-list\n4. An",
					"d another item.\n",
				},
				false,
			},
		},
		{
			&testInput{`
Emphasis, aka italics, with *asterisks* or _underscores_.

Strong emphasis, aka bold, with **asterisks** or __underscores__.

Combined emphasis with **asterisks and _underscores_**.

Strikethrough uses two tildes. ~~Scratch this.~~
`, 40, ""},
			&testOutput{
				[]string{
					"Emphasis, aka italics, with _asterisks_",
					" or _underscores_.",
					"Strong emphasis, aka bold, with ",
					"**asterisks** or **underscores**.",
					"Combined emphasis with ",
					"**asterisks and ****_underscores_**.",
					"Strikethrough uses two tildes. ",
					"~~Scratch this.~~",
				},
				false,
			},
		},
	}

	for i, tc := range testCases {
		tc := tc
		name := fmt.Sprintf("case_%d", i+1)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, ok := MarkdownSplit(tc.input.markdown, tc.input.max, tc.input.join)
			assert.Equal(t, tc.expected.chunks, result)
			assert.Equal(t, tc.expected.ok, ok)

			for _, cm := range result {
				correctLen := len(cm) <= tc.input.max
				assert.Truef(t, correctLen, "length is higher than max (%d)", len(cm))
			}
		})
	}
}
