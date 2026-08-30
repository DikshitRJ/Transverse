package templates

import (
	"fmt"
	"strings"
)

// GenerateTemplates creates starter boilerplate code with standard I/O and function structure
// for all 8 supported languages.
func GenerateTemplates(problemName, slug, topic string) map[string]string {
	cleanName := problemName
	if cleanName == "" {
		cleanName = "Problem"
	}
	funcName := toFunctionName(slug)
	if funcName == "" {
		funcName = "solve"
	}

	return map[string]string{
		"py":   generatePython(cleanName, funcName),
		"cpp":  generateCpp(cleanName, funcName),
		"java": generateJava(cleanName, funcName),
		"js":   generateJavaScript(cleanName, funcName),
		"go":   generateGo(cleanName, funcName),
		"rust": generateRust(cleanName, funcName),
		"c":    generateC(cleanName, funcName),
		"kt":   generateKotlin(cleanName, funcName),
	}
}

func toFunctionName(slug string) string {
	if slug == "" {
		return "solve"
	}
	parts := strings.Split(slug, "-")
	var res string
	for _, p := range parts {
		if len(p) > 0 {
			res += strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	if len(res) == 0 {
		return "solve"
	}
	// Lowercase first letter for camelCase
	return strings.ToLower(res[:1]) + res[1:]
}

func generatePython(title, funcName string) string {
	return fmt.Sprintf(`"""
Problem: %s
"""
import sys

def %s():
    # Read entire input from stdin
    input_data = sys.stdin.read().split()
    if not input_data:
        return
    
    # TODO: Parse input and implement solution
    # Example:
    # n = int(input_data[0])
    # print(result)
    pass

if __name__ == "__main__":
    %s()
`, title, funcName, funcName)
}

func generateCpp(title, funcName string) string {
	return fmt.Sprintf(`/*
 * Problem: %s
 */
#include <iostream>
#include <vector>
#include <string>
#include <algorithm>
#include <cmath>
#include <map>
#include <set>
#include <queue>

using namespace std;

void %s() {
    // Fast I/O is enabled in main()
    // TODO: Read input from cin and print output to cout
    // Example:
    // int n;
    // if (!(cin >> n)) return;
    // cout << n << "\n";
}

int main() {
    // Optimize standard I/O operations
    ios_base::sync_with_stdio(false);
    cin.tie(NULL);

    %s();
    return 0;
}
`, title, funcName, funcName)
}

func generateJava(title, funcName string) string {
	return fmt.Sprintf(`/*
 * Problem: %s
 */
import java.io.*;
import java.util.*;

public class Main {
    public static void %s(BufferedReader br, PrintWriter out) throws IOException {
        String line = br.readLine();
        if (line == null || line.trim().isEmpty()) {
            return;
        }

        // TODO: Read input and write output
        // StringTokenizer st = new StringTokenizer(line);
        // int n = Integer.parseInt(st.nextToken());
        // out.println(result);
    }

    public static void main(String[] args) throws IOException {
        BufferedReader br = new BufferedReader(new InputStreamReader(System.in));
        PrintWriter out = new PrintWriter(new BufferedOutputStream(System.out));

        %s(br, out);
        out.flush();
    }
}
`, title, funcName, funcName)
}

func generateJavaScript(title, funcName string) string {
	return fmt.Sprintf(`/**
 * Problem: %s
 */
const fs = require('fs');

function %s() {
    // Read all standard input
    const input = fs.readFileSync(0, 'utf-8').trim();
    if (!input) return;

    const lines = input.split(/\r?\n/);
    
    // TODO: Parse lines and output answer
    // console.log(result);
}

%s();
`, title, funcName, funcName)
}

func generateGo(title, funcName string) string {
	return fmt.Sprintf(`// Problem: %s
package main

import (
	"bufio"
	"fmt"
	"os"
)

func %s(scanner *bufio.Scanner, writer *bufio.Writer) {
	if !scanner.Scan() {
		return
	}
	// line := scanner.Text()

	// TODO: Process input and write to writer
	// fmt.Fprintln(writer, result)
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	%s(scanner, writer)
}
`, title, funcName, funcName)
}

func generateRust(title, funcName string) string {
	return fmt.Sprintf(`/*
 * Problem: %s
 */
use std::io::{self, Read};

fn %s() {
    let mut input = String::new();
    io::stdin().read_to_string(&mut input).unwrap();
    if input.trim().is_empty() {
        return;
    }

    // TODO: Parse input tokens and print answer
    // let mut tokens = input.split_whitespace();
    // println!("{}", result);
}

fn main() {
    %s();
}
`, title, funcName, funcName)
}

func generateC(title, funcName string) string {
	return fmt.Sprintf(`/*
 * Problem: %s
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

void %s() {
    // TODO: Read input from stdin and write to stdout
    // int n;
    // if (scanf("%%d", &n) != 1) return;
    // printf("%%d\n", result);
}

int main() {
    %s();
    return 0;
}
`, title, funcName, funcName)
}

func generateKotlin(title, funcName string) string {
	return fmt.Sprintf(`/*
 * Problem: %s
 */
import java.util.Scanner

fun %s(scanner: Scanner) {
    if (!scanner.hasNext()) return

    // TODO: Read input and print answer
    // val n = scanner.nextInt()
    // println(n)
}

fun main() {
    val scanner = Scanner(System.` + "`in`" + `)
    %s(scanner)
}
`, title, funcName, funcName)
}
