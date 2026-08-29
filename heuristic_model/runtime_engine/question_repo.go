package repository

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"velocity/internal/models"
)

type QuestionRepo struct {
	pool *pgxpool.Pool
}

func NewQuestionRepo(pool *pgxpool.Pool) *QuestionRepo {
	return &QuestionRepo{pool: pool}
}

func (r *QuestionRepo) GetByID(ctx context.Context, id string) (*models.Question, error) {
	var q models.Question
	var emb pgvector.Vector
	err := r.pool.QueryRow(ctx, `
		SELECT id, type, question_text, options, images, correct, subject, chapter,
		       chapter_group, difficulty, shift_date, source, exam_type,
		       glicko_rating, glicko_rd, timespent_avg_ms, percent_correct, attempt_count,
		       embedding
		FROM questions WHERE id = $1
	`, id).Scan(
		&q.ID, &q.Type, &q.QuestionText, &q.OptionsRaw, &q.ImagesRaw, &q.Correct,
		&q.Subject, &q.Chapter, &q.ChapterGroup, &q.Difficulty, &q.ShiftDate,
		&q.Source, &q.ExamType, &q.GlickoRating, &q.GlickoRD,
		&q.TimespentAvgMs, &q.PercentCorrect, &q.AttemptCount, &emb,
	)
	if err != nil {
		return nil, fmt.Errorf("question_repo: get by id %q: %w", id, err)
	}
	q.Embedding = emb
	return &q, nil
}

func (r *QuestionRepo) GetByIDWithoutEmbedding(ctx context.Context, id string) (*models.Question, error) {
	var q models.Question
	err := r.pool.QueryRow(ctx, `
		SELECT id, type, question_text, options, images, correct, subject, chapter,
		       chapter_group, difficulty, shift_date, source, exam_type,
		       glicko_rating, glicko_rd, timespent_avg_ms, percent_correct, attempt_count
		FROM questions WHERE id = $1
	`, id).Scan(
		&q.ID, &q.Type, &q.QuestionText, &q.OptionsRaw, &q.ImagesRaw, &q.Correct,
		&q.Subject, &q.Chapter, &q.ChapterGroup, &q.Difficulty, &q.ShiftDate,
		&q.Source, &q.ExamType, &q.GlickoRating, &q.GlickoRD,
		&q.TimespentAvgMs, &q.PercentCorrect, &q.AttemptCount,
	)
	if err != nil {
		return nil, fmt.Errorf("question_repo: get by id without emb %q: %w", id, err)
	}
	return &q, nil
}

func (r *QuestionRepo) GetByIDsWithoutEmbedding(ctx context.Context, ids []string) ([]models.Question, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, question_text, options, images, correct, subject, chapter,
		       chapter_group, difficulty, shift_date, source, exam_type,
		       glicko_rating, glicko_rd, timespent_avg_ms, percent_correct, attempt_count
		FROM questions WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("question_repo: get by ids: %w", err)
	}
	defer rows.Close()

	var result []models.Question
	for rows.Next() {
		var q models.Question
		if err := rows.Scan(
			&q.ID, &q.Type, &q.QuestionText, &q.OptionsRaw, &q.ImagesRaw, &q.Correct,
			&q.Subject, &q.Chapter, &q.ChapterGroup, &q.Difficulty, &q.ShiftDate,
			&q.Source, &q.ExamType, &q.GlickoRating, &q.GlickoRD,
			&q.TimespentAvgMs, &q.PercentCorrect, &q.AttemptCount,
		); err != nil {
			return nil, fmt.Errorf("question_repo: scan question: %w", err)
		}
		result = append(result, q)
	}
	return result, rows.Err()
}

func (r *QuestionRepo) GetByChapter(ctx context.Context, chapter string) ([]models.Question, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, question_text, options, images, correct, subject, chapter,
		       chapter_group, difficulty, shift_date, source, exam_type,
		       glicko_rating, glicko_rd, timespent_avg_ms, percent_correct, attempt_count
		FROM questions WHERE chapter = $1
	`, chapter)
	if err != nil {
		return nil, fmt.Errorf("question_repo: get by chapter: %w", err)
	}
	defer rows.Close()

	return r.scanQuestions(rows)
}

func (r *QuestionRepo) GetByChapters(ctx context.Context, chapters []string) ([]models.Question, error) {
	if len(chapters) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, question_text, options, images, correct, subject, chapter,
		       chapter_group, difficulty, shift_date, source, exam_type,
		       glicko_rating, glicko_rd, timespent_avg_ms, percent_correct, attempt_count,
		       embedding
		FROM questions WHERE chapter = ANY($1)
	`, chapters)
	if err != nil {
		return nil, fmt.Errorf("question_repo: get by chapters: %w", err)
	}
	defer rows.Close()

	return r.scanQuestionsWithEmb(rows)
}

func (r *QuestionRepo) GetByChaptersFiltered(ctx context.Context, chapters, years, examTypes []string) ([]models.Question, error) {
	if len(chapters) == 0 {
		return nil, nil
	}
	query := `SELECT id, type, question_text, options, images, correct, subject, chapter,
		chapter_group, difficulty, shift_date, source, exam_type,
		glicko_rating, glicko_rd, timespent_avg_ms, percent_correct, attempt_count,
		embedding
		FROM questions WHERE chapter = ANY($1)`
	args := []interface{}{chapters}
	argIdx := 2

	if len(years) > 0 {
		query += fmt.Sprintf(` AND shift_date ~ ANY($%d)`, argIdx)
		likeYears := make([]string, len(years))
		for i, y := range years {
			likeYears[i] = fmt.Sprintf("^%s", y)
		}
		args = append(args, likeYears)
		argIdx++
	}
	if len(examTypes) > 0 {
		query += fmt.Sprintf(` AND exam_type = ANY($%d)`, argIdx)
		args = append(args, examTypes)
		argIdx++
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("question_repo: get by chapters filtered: %w", err)
	}
	defer rows.Close()

	return r.scanQuestionsWithEmb(rows)
}

func (r *QuestionRepo) GetByChaptersFilteredLight(ctx context.Context, chapters []string, years, examTypes []string) ([]models.Question, error) {
	if len(chapters) == 0 {
		return nil, nil
	}
	query := `
		SELECT id, type, question_text, options, images, correct, subject, chapter,
		       chapter_group, difficulty, shift_date, source, exam_type,
		       glicko_rating, glicko_rd, timespent_avg_ms, percent_correct, attempt_count
		FROM questions WHERE chapter = ANY($1)`
	args := []interface{}{chapters}
	argIdx := 2
	if len(examTypes) > 0 {
		query += fmt.Sprintf(" AND exam_type = ANY($%d)", argIdx)
		args = append(args, examTypes)
		argIdx++
	}
	if len(years) > 0 {
		query += fmt.Sprintf(" AND SUBSTRING(shift_date FROM 1 FOR 4) = ANY($%d)", argIdx)
		args = append(args, years)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("question_repo: get by chapters light: %w", err)
	}
	defer rows.Close()

	var result []models.Question
	for rows.Next() {
		var q models.Question
		if err := rows.Scan(
			&q.ID, &q.Type, &q.QuestionText, &q.OptionsRaw, &q.ImagesRaw, &q.Correct,
			&q.Subject, &q.Chapter, &q.ChapterGroup, &q.Difficulty, &q.ShiftDate,
			&q.Source, &q.ExamType, &q.GlickoRating, &q.GlickoRD,
			&q.TimespentAvgMs, &q.PercentCorrect, &q.AttemptCount,
		); err != nil {
			return nil, fmt.Errorf("question_repo: scan question: %w", err)
		}
		result = append(result, q)
	}
	return result, rows.Err()
}

func (r *QuestionRepo) GetChaptersTotalCountFiltered(ctx context.Context, chapters []string, examType string) (int, error) {
	if len(chapters) == 0 {
		return 0, nil
	}
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM questions
		WHERE chapter = ANY($1) AND ($2 = '' OR exam_type = $2)
	`, chapters, examType).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("question_repo: get chapters total count filtered: %w", err)
	}
	return count, nil
}

func (r *QuestionRepo) GetChapterInfo(ctx context.Context) ([]models.ChapterInfo, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT chapter, COALESCE(chapter_group, ''), subject,
		       COUNT(*) AS total,
		       COUNT(*) FILTER (WHERE glicko_rating < 1350) AS very_easy,
		       COUNT(*) FILTER (WHERE glicko_rating >= 1350 AND glicko_rating < 1450) AS easy,
		       COUNT(*) FILTER (WHERE glicko_rating >= 1450 AND glicko_rating < 1550) AS medium,
		       COUNT(*) FILTER (WHERE glicko_rating >= 1550 AND glicko_rating < 1700) AS hard,
		       COUNT(*) FILTER (WHERE glicko_rating >= 1700) AS very_hard,
		       ARRAY_AGG(DISTINCT exam_type) FILTER (WHERE exam_type IS NOT NULL AND exam_type != '') AS exam_types
		FROM questions
		GROUP BY chapter, chapter_group, subject
		ORDER BY subject, chapter
	`)
	if err != nil {
		return nil, fmt.Errorf("question_repo: get chapter info: %w", err)
	}
	defer rows.Close()

	var result []models.ChapterInfo
	for rows.Next() {
		var ci models.ChapterInfo
		if err := rows.Scan(&ci.Chapter, &ci.ChapterGroup, &ci.Subject,
			&ci.TotalQuestions, &ci.VeryEasyCount, &ci.EasyCount,
			&ci.MediumCount, &ci.HardCount, &ci.VeryHardCount,
			&ci.ExamTypes); err != nil {
			return nil, fmt.Errorf("question_repo: scan chapter info: %w", err)
		}
		result = append(result, ci)
	}
	return result, rows.Err()
}

func (r *QuestionRepo) GetAvailableYears(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT SUBSTRING(shift_date FROM 1 FOR 4) AS year
		FROM questions WHERE shift_date IS NOT NULL AND shift_date != ''
		ORDER BY year DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("question_repo: get available years: %w", err)
	}
	defer rows.Close()

	var years []string
	for rows.Next() {
		var y string
		if err := rows.Scan(&y); err != nil {
			return nil, err
		}
		years = append(years, y)
	}
	return years, nil
}

func (r *QuestionRepo) GetAvailableExamTypes(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT exam_type FROM questions
		WHERE exam_type IS NOT NULL AND exam_type != ''
		ORDER BY exam_type
	`)
	if err != nil {
		return nil, fmt.Errorf("question_repo: get available exam types: %w", err)
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return types, nil
}

func (r *QuestionRepo) ResolveScopeByDB(ctx context.Context, subjects, chapterGroups, chapters, years, examTypes []string) ([]string, error) {
	if len(chapters) > 0 {
		return chapters, nil
	}

	var where []string
	var args []interface{}
	argIdx := 1

	if len(subjects) > 0 {
		where = append(where, fmt.Sprintf("subject = ANY($%d)", argIdx))
		args = append(args, subjects)
		argIdx++
	}
	if len(chapterGroups) > 0 {
		where = append(where, fmt.Sprintf("chapter_group = ANY($%d)", argIdx))
		args = append(args, chapterGroups)
		argIdx++
	}

	query := `SELECT DISTINCT chapter FROM questions`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY chapter"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("question_repo: resolve scope: %w", err)
	}
	defer rows.Close()

	var resolved []string
	for rows.Next() {
		var ch string
		if err := rows.Scan(&ch); err != nil {
			return nil, err
		}
		resolved = append(resolved, ch)
	}
	return resolved, rows.Err()
}

func (r *QuestionRepo) UpdateQuestionStats(ctx context.Context, questionID string, timeTakenMs int64, isCorrect bool) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE questions SET
			attempt_count = attempt_count + 1,
			timespent_avg_ms = (timespent_avg_ms * GREATEST(attempt_count, 1) + $2) / (attempt_count + 1),
			percent_correct = ((percent_correct::float8 * attempt_count) + CASE WHEN $3 THEN 100 ELSE 0 END) / (attempt_count + 1)
		WHERE id = $1
	`, questionID, timeTakenMs, isCorrect)
	if err != nil {
		return fmt.Errorf("question_repo: update stats: %w", err)
	}
	return nil
}

func (r *QuestionRepo) GetChaptersTotalCount(ctx context.Context, chapters []string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM questions WHERE chapter = ANY($1)`, chapters).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("question_repo: get chapters total count: %w", err)
	}
	return count, nil
}

func (r *QuestionRepo) scanQuestions(rows pgx.Rows) ([]models.Question, error) {
	var result []models.Question
	for rows.Next() {
		var q models.Question
		if err := rows.Scan(
			&q.ID, &q.Type, &q.QuestionText, &q.OptionsRaw, &q.ImagesRaw, &q.Correct,
			&q.Subject, &q.Chapter, &q.ChapterGroup, &q.Difficulty, &q.ShiftDate,
			&q.Source, &q.ExamType, &q.GlickoRating, &q.GlickoRD,
			&q.TimespentAvgMs, &q.PercentCorrect, &q.AttemptCount,
		); err != nil {
			return nil, fmt.Errorf("question_repo: scan question: %w", err)
		}
		result = append(result, q)
	}
	return result, rows.Err()
}

func (r *QuestionRepo) scanQuestionsWithEmb(rows pgx.Rows) ([]models.Question, error) {
	var result []models.Question
	for rows.Next() {
		var q models.Question
		var emb pgvector.Vector
		if err := rows.Scan(
			&q.ID, &q.Type, &q.QuestionText, &q.OptionsRaw, &q.ImagesRaw, &q.Correct,
			&q.Subject, &q.Chapter, &q.ChapterGroup, &q.Difficulty, &q.ShiftDate,
			&q.Source, &q.ExamType, &q.GlickoRating, &q.GlickoRD,
			&q.TimespentAvgMs, &q.PercentCorrect, &q.AttemptCount,
			&emb,
		); err != nil {
			return nil, fmt.Errorf("question_repo: scan question: %w", err)
		}
		q.Embedding = emb
		result = append(result, q)
	}
	return result, rows.Err()
}

// FindSimilarByChapter returns top-N questions from a chapter ordered by cosine
// similarity to the given embedding, using the pgvector HNSW index.
// excludeID (optional) excludes a specific question (e.g. the current one).
func (r *QuestionRepo) FindSimilarByChapter(ctx context.Context, emb pgvector.Vector, chapter, excludeID string, limit int) ([]models.Question, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, question_text, options, images, correct, subject, chapter,
		       chapter_group, difficulty, shift_date, source, exam_type,
		       glicko_rating, glicko_rd, timespent_avg_ms, percent_correct, attempt_count,
		       embedding
		FROM questions
		WHERE chapter = $1
		  AND embedding IS NOT NULL
		  AND ($2 = '' OR id != $2)
		ORDER BY embedding <=> $3
		LIMIT $4
	`, chapter, excludeID, emb, limit)
	if err != nil {
		return nil, fmt.Errorf("question_repo: find similar by chapter: %w", err)
	}
	defer rows.Close()
	return r.scanQuestionsWithEmb(rows)
}

// FindSimilarByChapters returns top-N questions across multiple chapters ordered
// by cosine similarity to the given embedding, using the pgvector HNSW index.
func (r *QuestionRepo) FindSimilarByChapters(ctx context.Context, emb pgvector.Vector, chapters []string, excludeID string, limit int) ([]models.Question, error) {
	if len(chapters) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, question_text, options, images, correct, subject, chapter,
		       chapter_group, difficulty, shift_date, source, exam_type,
		       glicko_rating, glicko_rd, timespent_avg_ms, percent_correct, attempt_count,
		       embedding
		FROM questions
		WHERE chapter = ANY($1)
		  AND embedding IS NOT NULL
		  AND ($2 = '' OR id != $2)
		ORDER BY embedding <=> $3
		LIMIT $4
	`, chapters, excludeID, emb, limit)
	if err != nil {
		return nil, fmt.Errorf("question_repo: find similar by chapters: %w", err)
	}
	defer rows.Close()
	return r.scanQuestionsWithEmb(rows)
}

func SlugToDisplayName(slug string) string {
	if slug == "" {
		return ""
	}
	words := strings.FieldsFunc(slug, func(r rune) bool { return r == '-' || r == '_' })
	for i, w := range words {
		if len(w) > 0 {
			runes := []rune(w)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}
