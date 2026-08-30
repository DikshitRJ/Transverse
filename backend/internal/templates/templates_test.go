package templates

import (
	"strings"
	"testing"
)

func TestGenerateTemplates_AllLanguages(t *testing.T) {
	temps := GenerateTemplates("Two Sum", "two-sum", "hash-table")

	expectedLangs := []string{"py", "cpp", "java", "js", "go", "rust", "c", "kt"}
	for _, lang := range expectedLangs {
		code, ok := temps[lang]
		if !ok {
			t.Errorf("expected template for language %q to exist", lang)
			continue
		}
		if len(strings.TrimSpace(code)) == 0 {
			t.Errorf("template for language %q should not be empty", lang)
		}
	}

	// Python verification
	py := temps["py"]
	if !strings.Contains(py, "def twoSum():") {
		t.Errorf("python template missing function twoSum: %s", py)
	}
	if !strings.Contains(py, "sys.stdin.read()") {
		t.Errorf("python template missing sys.stdin.read: %s", py)
	}

	// C++ verification
	cpp := temps["cpp"]
	if !strings.Contains(cpp, "void twoSum()") {
		t.Errorf("cpp template missing function twoSum: %s", cpp)
	}
	if !strings.Contains(cpp, "ios_base::sync_with_stdio(false)") {
		t.Errorf("cpp template missing fast I/O: %s", cpp)
	}

	// Java verification
	java := temps["java"]
	if !strings.Contains(java, "public static void twoSum(BufferedReader br, PrintWriter out)") {
		t.Errorf("java template missing twoSum function: %s", java)
	}
	if !strings.Contains(java, "public class Main") {
		t.Errorf("java template missing Main class: %s", java)
	}

	// Go verification
	goCode := temps["go"]
	if !strings.Contains(goCode, "func twoSum(scanner *bufio.Scanner, writer *bufio.Writer)") {
		t.Errorf("go template missing twoSum: %s", goCode)
	}

	// JS verification
	js := temps["js"]
	if !strings.Contains(js, "function twoSum()") {
		t.Errorf("js template missing twoSum: %s", js)
	}
}

func TestToFunctionName(t *testing.T) {
	cases := []struct {
		slug     string
		expected string
	}{
		{"two-sum", "twoSum"},
		{"longest-common-subsequence", "longestCommonSubsequence"},
		{"", "solve"},
		{"single", "single"},
	}

	for _, tc := range cases {
		actual := toFunctionName(tc.slug)
		if actual != tc.expected {
			t.Errorf("toFunctionName(%q) = %q; expected %q", tc.slug, actual, tc.expected)
		}
	}
}
