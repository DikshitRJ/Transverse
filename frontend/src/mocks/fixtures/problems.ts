/**
 * ~40 mock problems shaped exactly like `models.ProblemPayload`
 * (backend/internal/models/dto.go) — tags[], subtopic, avg_time_ms,
 * contest_id, and NO glicko_rating/status fields (a real drift from
 * `openapi.yaml`, see plan.md §0.2). `templates` keys match the real
 * backend's 8 languages (`py`,`cpp`,`java`,`js`,`go`,`rust`,`c`,`kt`).
 *
 * Statements are plain HTML strings, matching what the real scrapers
 * (internal/scraper/*.go) actually produce from LeetCode/Codeforces pages —
 * treat as UNTRUSTED when rendering (react-markdown + rehype-sanitize /
 * rehype-raw, per plan.md's stack table). Never `dangerouslySetInnerHTML`
 * these without sanitizing first, even though this copy is fixture data —
 * build the pipeline as if it were live scraped HTML, because it will be.
 */

import type { ProblemPayload, TestCase } from "@/lib/api/types";
import { generateTemplates } from "./languages";

const SOURCES = ["leetcode", "codeforces", "atcoder", "cses"] as const;
const DIFFICULTIES = ["easy", "medium", "hard", "expert"] as const;

interface HeroProblem {
  slug: string;
  name: string;
  topic: string;
  subtopic: string;
  source: (typeof SOURCES)[number];
  difficulty: (typeof DIFFICULTIES)[number];
  tags: string[];
  statement: string;
  testCases: TestCase[];
}

function statementShell(opts: {
  title: string;
  body: string;
  constraints: string[];
  examples: { input: string; output: string; explanation?: string }[];
}): string {
  const examples = opts.examples
    .map(
      (ex, i) => `
<h3>Example ${i + 1}</h3>
<pre><code>Input: ${ex.input}
Output: ${ex.output}${ex.explanation ? `\nExplanation: ${ex.explanation}` : ""}</code></pre>`,
    )
    .join("\n");
  const constraints = opts.constraints.map((c) => `<li><code>${c}</code></li>`).join("\n");
  return `<h2>${opts.title}</h2>\n<p>${opts.body}</p>\n${examples}\n<h3>Constraints</h3>\n<ul>\n${constraints}\n</ul>`;
}

const HEROES: HeroProblem[] = [
  {
    slug: "two-sum",
    name: "Two Sum",
    topic: "arrays-hashing",
    subtopic: "hash table",
    source: "leetcode",
    difficulty: "easy",
    tags: ["array", "hash table"],
    statement: statementShell({
      title: "Two Sum",
      body: "Given an array of integers <code>nums</code> and an integer <code>target</code>, return the indices of the two numbers such that they add up to <code>target</code>. You may assume exactly one solution exists, and you may not use the same element twice.",
      constraints: ["2 <= nums.length <= 10^4", "-10^9 <= nums[i] <= 10^9", "-10^9 <= target <= 10^9"],
      examples: [
        { input: "nums = [2,7,11,15], target = 9", output: "[0,1]", explanation: "nums[0] + nums[1] == 9" },
        { input: "nums = [3,2,4], target = 6", output: "[1,2]" },
      ],
    }),
    testCases: [
      { input: "4\n2 7 11 15\n9", output: "0 1" },
      { input: "3\n3 2 4\n6", output: "1 2" },
      { input: "2\n3 3\n6", output: "0 1", is_hidden: true },
    ],
  },
  {
    slug: "valid-parentheses",
    name: "Valid Parentheses",
    topic: "stack-queues",
    subtopic: "stack",
    source: "leetcode",
    difficulty: "easy",
    tags: ["string", "stack"],
    statement: statementShell({
      title: "Valid Parentheses",
      body: "Given a string <code>s</code> containing just the characters <code>'(' ')' '{' '}' '[' ']'</code>, determine if the input string is valid — every opening bracket must be closed by the same type of bracket, in the correct order.",
      constraints: ["1 <= s.length <= 10^4", "s consists only of bracket characters"],
      examples: [
        { input: 's = "()[]{}"', output: "true" },
        { input: 's = "(]"', output: "false" },
      ],
    }),
    testCases: [
      { input: "()[]{}", output: "true" },
      { input: "(]", output: "false" },
      { input: "([)]", output: "false", is_hidden: true },
    ],
  },
  {
    slug: "binary-search",
    name: "Binary Search",
    topic: "binary-search",
    subtopic: "classic binary search",
    source: "leetcode",
    difficulty: "easy",
    tags: ["array", "binary search"],
    statement: statementShell({
      title: "Binary Search",
      body: "Given a sorted array of distinct integers <code>nums</code> and a <code>target</code>, return the index of <code>target</code> in <code>nums</code>, or <code>-1</code> if it is not present. Must run in O(log n) time.",
      constraints: ["1 <= nums.length <= 10^4", "nums is sorted in strictly ascending order"],
      examples: [{ input: "nums = [-1,0,3,5,9,12], target = 9", output: "4" }],
    }),
    testCases: [
      { input: "6\n-1 0 3 5 9 12\n9", output: "4" },
      { input: "6\n-1 0 3 5 9 12\n2", output: "-1" },
    ],
  },
  {
    slug: "merge-two-sorted-lists",
    name: "Merge Two Sorted Lists",
    topic: "linked-list",
    subtopic: "linked list merge",
    source: "leetcode",
    difficulty: "easy",
    tags: ["linked list", "recursion"],
    statement: statementShell({
      title: "Merge Two Sorted Lists",
      body: "You are given the heads of two sorted linked lists <code>list1</code> and <code>list2</code>. Merge the two lists into one sorted list by splicing together the nodes of the first two lists, and return the head of the merged list.",
      constraints: ["The number of nodes in both lists is in the range [0, 50]", "-100 <= Node.val <= 100"],
      examples: [{ input: "list1 = [1,2,4], list2 = [1,3,4]", output: "[1,1,2,3,4,4]" }],
    }),
    testCases: [
      { input: "1 2 4\n1 3 4", output: "1 1 2 3 4 4" },
      { input: "\n\n", output: "" },
    ],
  },
  {
    slug: "longest-substring-without-repeating-characters",
    name: "Longest Substring Without Repeating Characters",
    topic: "sliding-window",
    subtopic: "variable window",
    source: "leetcode",
    difficulty: "medium",
    tags: ["hash table", "string", "sliding window"],
    statement: statementShell({
      title: "Longest Substring Without Repeating Characters",
      body: "Given a string <code>s</code>, find the length of the longest substring without repeating characters.",
      constraints: ["0 <= s.length <= 5 * 10^4"],
      examples: [
        { input: 's = "abcabcbb"', output: "3", explanation: 'The answer is "abc"' },
        { input: 's = "bbbbb"', output: "1" },
      ],
    }),
    testCases: [
      { input: "abcabcbb", output: "3" },
      { input: "bbbbb", output: "1" },
      { input: "pwwkew", output: "3", is_hidden: true },
    ],
  },
  {
    slug: "course-schedule",
    name: "Course Schedule",
    topic: "topological-sort",
    subtopic: "cycle detection",
    source: "leetcode",
    difficulty: "medium",
    tags: ["graph", "topological sort", "dfs"],
    statement: statementShell({
      title: "Course Schedule",
      body: "There are <code>numCourses</code> courses labeled 0 to numCourses-1. Given the prerequisite pairs, determine if it's possible to finish all courses (i.e., is the prerequisite graph a DAG).",
      constraints: ["1 <= numCourses <= 2000", "0 <= prerequisites.length <= 5000"],
      examples: [
        { input: "numCourses = 2, prerequisites = [[1,0]]", output: "true" },
        { input: "numCourses = 2, prerequisites = [[1,0],[0,1]]", output: "false" },
      ],
    }),
    testCases: [
      { input: "2\n1\n1 0", output: "true" },
      { input: "2\n2\n1 0\n0 1", output: "false" },
    ],
  },
  {
    slug: "climbing-stairs",
    name: "Climbing Stairs",
    topic: "dynamic-programming",
    subtopic: "1d dp",
    source: "leetcode",
    difficulty: "easy",
    tags: ["dynamic programming", "math"],
    statement: statementShell({
      title: "Climbing Stairs",
      body: "You are climbing a staircase with <code>n</code> steps. Each time you can climb 1 or 2 steps. In how many distinct ways can you climb to the top?",
      constraints: ["1 <= n <= 45"],
      examples: [
        { input: "n = 2", output: "2" },
        { input: "n = 3", output: "3" },
      ],
    }),
    testCases: [
      { input: "2", output: "2" },
      { input: "5", output: "8", is_hidden: true },
    ],
  },
  {
    slug: "kth-largest-element",
    name: "Kth Largest Element in an Array",
    topic: "heaps-priority-queues",
    subtopic: "quickselect / heap",
    source: "leetcode",
    difficulty: "medium",
    tags: ["heap", "divide and conquer", "quickselect"],
    statement: statementShell({
      title: "Kth Largest Element in an Array",
      body: "Given an integer array <code>nums</code> and an integer <code>k</code>, return the kth largest element in the array (kth largest by value, not kth distinct).",
      constraints: ["1 <= k <= nums.length <= 10^5"],
      examples: [{ input: "nums = [3,2,1,5,6,4], k = 2", output: "5" }],
    }),
    testCases: [
      { input: "6\n3 2 1 5 6 4\n2", output: "5" },
      { input: "9\n3 2 3 1 2 4 5 5 6\n4", output: "4", is_hidden: true },
    ],
  },
  {
    slug: "1-a-plus-b",
    name: "A+B Problem",
    topic: "foundations",
    subtopic: "implementation",
    source: "codeforces",
    difficulty: "easy",
    tags: ["implementation", "math"],
    statement: statementShell({
      title: "A+B Problem",
      body: "You are given two integers <code>a</code> and <code>b</code>. Print their sum.",
      constraints: ["-10^9 <= a, b <= 10^9"],
      examples: [{ input: "1 2", output: "3" }],
    }),
    testCases: [
      { input: "1 2", output: "3" },
      { input: "-5 5", output: "0" },
    ],
  },
  {
    slug: "watermelon",
    name: "Watermelon",
    topic: "math-number-theory",
    subtopic: "parity",
    source: "codeforces",
    difficulty: "easy",
    tags: ["brute force", "math"],
    statement: statementShell({
      title: "Watermelon",
      body: "Given a watermelon weighing <code>w</code> kilos, determine whether it's possible to split it into two parts, each an even, positive integer number of kilos.",
      constraints: ["1 <= w <= 100"],
      examples: [{ input: "8", output: "YES" }],
    }),
    testCases: [
      { input: "8", output: "YES" },
      { input: "3", output: "NO" },
    ],
  },
];

function seededPick<T>(arr: readonly T[], seed: number): T {
  return arr[seed % arr.length]!;
}

function buildStatement(name: string, topic: string): string {
  return statementShell({
    title: name,
    body: `Solve <strong>${name}</strong>. This problem exercises the <em>${topic.replace(/-/g, " ")}</em> pattern — read the constraints carefully and aim for the optimal time complexity for this topic before submitting.`,
    constraints: ["1 <= n <= 10^5", "-10^9 <= values[i] <= 10^9"],
    examples: [{ input: "see samples", output: "see samples" }],
  });
}

const FILLER_TOPICS = [
  "arrays-hashing",
  "two-pointers",
  "sliding-window",
  "stack-queues",
  "binary-search",
  "sorting-searching",
  "linked-list",
  "trees",
  "tries",
  "heaps-priority-queues",
  "backtracking",
  "graphs",
  "dynamic-programming",
  "greedy",
  "bit-manipulation",
];

const FILLER_NAME_PARTS = [
  "Traversal",
  "Subsequence",
  "Partition",
  "Rotation",
  "Merge",
  "Interval",
  "Cycle Detection",
  "Path Sum",
  "Window",
  "Frequency",
  "Distance",
  "Reachability",
  "Reordering",
  "Minimum Cost",
  "Counting",
];

function buildFillerProblems(count: number): ProblemPayload[] {
  const out: ProblemPayload[] = [];
  for (let i = 0; i < count; i++) {
    const topic = seededPick(FILLER_TOPICS, i);
    const namePart = seededPick(FILLER_NAME_PARTS, i * 3 + 1);
    const source = seededPick(SOURCES, i * 5 + 2);
    const difficulty = seededPick(DIFFICULTIES, i * 7 + 3);
    const name = `${topic
      .split("-")
      .map((p) => p[0]!.toUpperCase() + p.slice(1))
      .join(" ")} ${namePart}`;
    const slug = `${topic}-${namePart.toLowerCase().replace(/\s+/g, "-")}-${i}`;
    const id = `p-fill-${String(i).padStart(3, "0")}`;
    out.push({
      id,
      source,
      name,
      url: `https://example.com/${source}/${slug}`,
      slug,
      contest_id: source === "codeforces" ? String(1400 + i) : undefined,
      tags: [topic.replace(/-/g, " "), namePart.toLowerCase()],
      topic,
      subtopic: namePart.toLowerCase(),
      difficulty_label: difficulty,
      solve_rate: Math.round((0.25 + ((i * 13) % 60) / 100) * 1000) / 1000,
      avg_time_ms: 4000 + ((i * 977) % 20000),
      statement: buildStatement(name, topic),
      test_cases: [
        { input: `${i}\n${i + 1} ${i + 2}`, output: `${i * 2 + 3}` },
        { input: `${i + 10}\n${i + 5} ${i + 6}`, output: `${i * 2 + 11}`, is_hidden: true },
      ],
      templates: generateTemplates(name, slug),
    });
  }
  return out;
}

const heroProblems: ProblemPayload[] = HEROES.map((h, i) => ({
  id: `p-hero-${String(i).padStart(3, "0")}`,
  source: h.source,
  name: h.name,
  url: `https://${h.source}.com/problems/${h.slug}`,
  slug: h.slug,
  contest_id: h.source === "codeforces" ? "4" : undefined,
  tags: h.tags,
  topic: h.topic,
  subtopic: h.subtopic,
  difficulty_label: h.difficulty,
  solve_rate: [0.72, 0.58, 0.81, 0.69, 0.44, 0.39, 0.76, 0.41, 0.9, 0.65][i] ?? 0.5,
  avg_time_ms: [180000, 240000, 150000, 300000, 420000, 480000, 200000, 360000, 60000, 90000][i] ?? 300000,
  statement: h.statement,
  test_cases: h.testCases,
  templates: generateTemplates(h.name, h.slug),
}));

export const PROBLEMS: ProblemPayload[] = [...heroProblems, ...buildFillerProblems(30)];

export function getProblemById(id: string): ProblemPayload | undefined {
  return PROBLEMS.find((p) => p.id === id);
}

export function getProblemBySlug(slug: string): ProblemPayload | undefined {
  return PROBLEMS.find((p) => p.slug === slug);
}
