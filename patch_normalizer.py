import re

with open("backend/pipeline/seed/normalizer.go", "r") as f:
    content = f.read()

# Make sure we import encoding/json
if '"encoding/json"' not in content:
    content = content.replace('"strings"', '"encoding/json"\n\t"strings"')

# Update Normalize struct instantiation
replacement = """		GlickoVolatility: 0.06,
		EmbedText:        embedText,
		Statement:        raw.Statement,
	}
	
	// Convert test cases to JSON
	type tc struct {
		Input  string `json:"input"`
		Output string `json:"output"`
	}
	var tcs []tc
	for i := range raw.InputTestcases {
	    if i < len(raw.OutputTestcases) {
	        tcs = append(tcs, tc{Input: raw.InputTestcases[i], Output: raw.OutputTestcases[i]})
	    }
	}
	if b, err := json.Marshal(tcs); err == nil {
	    ret.TestCases = b
	} else {
	    ret.TestCases = []byte("[]")
	}
	
	return ret
}"""

# Find the end of the return NormalizedProblem struct
content = re.sub(r'GlickoVolatility:\s*0\.06,\s*EmbedText:\s*embedText,\s*\}', replacement, content)
content = content.replace("return NormalizedProblem{", "ret := NormalizedProblem{")

with open("backend/pipeline/seed/normalizer.go", "w") as f:
    f.write(content)
