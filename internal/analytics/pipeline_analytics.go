package analytics

import (
	"fmt"
	"math"
	"time"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

// ─── Output types ─────────────────────────────────────────────────────────────

type PipelineAnalytics struct {
	TotalApplications    int                `json:"totalApplications"`
	ResponseRate         float64            `json:"responseRate"`
	AverageAtsScore      float64            `json:"averageAtsScore"`
	InterviewCount       int                `json:"interviewCount"`
	OfferCount           int                `json:"offerCount"`
	GhostedCount         int                `json:"ghostedCount"`
	ApplicationsThisWeek int                `json:"applicationsThisWeek"`
	ScoreRangeBuckets    []ScoreRangeBucket `json:"scoreRangeBuckets"`
	BestResumeVersion    *BestResume        `json:"bestResumeVersion"`
	StageDistribution    []StageCount       `json:"stageDistribution"`
	WeeklyActivity       []WeeklyPoint      `json:"weeklyActivity"`
	CoachInsight         string             `json:"coachInsight"`
}

type ScoreRangeBucket struct {
	Label        string  `json:"label"`
	ResponseRate float64 `json:"responseRate"`
	Count        int     `json:"count"`
}

type BestResume struct {
	ResumeId         string  `json:"resumeId"`
	ResumeName       string  `json:"resumeName"`
	ResponseRate     float64 `json:"responseRate"`
	ApplicationCount int     `json:"applicationCount"`
}

type StageCount struct {
	Stage string `json:"stage"`
	Count int    `json:"count"`
}

type WeeklyPoint struct {
	WeekLabel        string `json:"weekLabel"`
	ApplicationCount int    `json:"applicationCount"`
	ResponseCount    int    `json:"responseCount"`
}

// ─── Compute ──────────────────────────────────────────────────────────────────

// Compute derives all analytics metrics from jobs in memory — no extra DB calls.
func Compute(jobs []domain.PipelineJob) PipelineAnalytics {
	if len(jobs) == 0 {
		return PipelineAnalytics{
			ScoreRangeBuckets: []ScoreRangeBucket{},
			StageDistribution: []StageCount{},
			WeeklyActivity:    buildWeeklyActivity(nil),
			CoachInsight:      defaultInsight(),
		}
	}

	total := len(jobs)

	// Responded stages
	respondedStages := map[domain.PipelineJobStage]bool{
		domain.StageInterview: true,
		domain.StageOffer:     true,
		domain.StageRejected:  true,
	}

	responded := 0
	atsSum := 0.0
	atsCount := 0
	interviewCount := 0
	offerCount := 0
	ghostedCount := 0

	stageMap := map[string]int{}

	for _, j := range jobs {
		if respondedStages[j.Stage] && !j.IsGhosted {
			responded++
		}
		if j.AtsScore > 0 {
			atsSum += float64(j.AtsScore)
			atsCount++
		}
		if j.Stage == domain.StageInterview {
			interviewCount++
		}
		if j.Stage == domain.StageOffer {
			offerCount++
		}
		if j.IsGhosted {
			ghostedCount++
		}
		stageMap[string(j.Stage)]++
	}

	responseRate := round1(float64(responded) / float64(total) * 100)

	averageAtsScore := 0.0
	if atsCount > 0 {
		averageAtsScore = round1(atsSum / float64(atsCount))
	}

	// Applications this week
	thisWeekMonday := isoWeekMonday(time.Now())
	thisWeekCount := 0
	for _, j := range jobs {
		if !j.CreatedAt.Before(thisWeekMonday) {
			thisWeekCount++
		}
	}

	// Score range buckets
	type bucket struct {
		label string
		min   int
		max   int
	}
	bucketDefs := []bucket{
		{"90-100", 90, 100},
		{"75-89", 75, 89},
		{"60-74", 60, 74},
		{"<60", 0, 59},
	}

	var scoreBuckets []ScoreRangeBucket
	for _, b := range bucketDefs {
		var group []domain.PipelineJob
		for _, j := range jobs {
			if j.AtsScore >= b.min && j.AtsScore <= b.max {
				group = append(group, j)
			}
		}
		if len(group) == 0 {
			continue
		}
		rr := responseRateOf(group, respondedStages)
		scoreBuckets = append(scoreBuckets, ScoreRangeBucket{
			Label:        b.label,
			ResponseRate: rr,
			Count:        len(group),
		})
	}

	// Best resume version
	var bestResume *BestResume
	type resumeGroup struct {
		count     int
		responded int
	}
	resumeMap := map[string]*resumeGroup{}
	for _, j := range jobs {
		if j.ResumeID == "" {
			continue
		}
		g, ok := resumeMap[j.ResumeID]
		if !ok {
			g = &resumeGroup{}
			resumeMap[j.ResumeID] = g
		}
		g.count++
		if respondedStages[j.Stage] && !j.IsGhosted {
			g.responded++
		}
	}
	bestRR := -1.0
	for rid, g := range resumeMap {
		if g.count < 2 {
			continue
		}
		rr := float64(g.responded) / float64(g.count) * 100
		if rr > bestRR {
			bestRR = rr
			bestResume = &BestResume{
				ResumeId:         rid,
				ResumeName:       rid,
				ResponseRate:     round1(rr),
				ApplicationCount: g.count,
			}
		}
	}

	// Stage distribution
	stageOrder := []string{"wishlist", "applied", "interview", "offer", "rejected"}
	var stageDist []StageCount
	for _, s := range stageOrder {
		if c, ok := stageMap[s]; ok {
			stageDist = append(stageDist, StageCount{Stage: s, Count: c})
		}
	}

	// Weekly activity — last 4 ISO weeks
	weeklyActivity := buildWeeklyActivity(jobs)

	// Coach insight
	coachInsight := computeInsight(scoreBuckets, responseRate, total, bestResume, thisWeekCount)

	return PipelineAnalytics{
		TotalApplications:    total,
		ResponseRate:         responseRate,
		AverageAtsScore:      averageAtsScore,
		InterviewCount:       interviewCount,
		OfferCount:           offerCount,
		GhostedCount:         ghostedCount,
		ApplicationsThisWeek: thisWeekCount,
		ScoreRangeBuckets:    scoreBuckets,
		BestResumeVersion:    bestResume,
		StageDistribution:    stageDist,
		WeeklyActivity:       weeklyActivity,
		CoachInsight:         coachInsight,
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

// isoWeekMonday returns midnight UTC of the Monday starting the ISO week
// that contains t.
func isoWeekMonday(t time.Time) time.Time {
	t = t.UTC()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday → 7
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

func responseRateOf(jobs []domain.PipelineJob, respondedStages map[domain.PipelineJobStage]bool) float64 {
	if len(jobs) == 0 {
		return 0
	}
	r := 0
	for _, j := range jobs {
		if respondedStages[j.Stage] && !j.IsGhosted {
			r++
		}
	}
	return round1(float64(r) / float64(len(jobs)) * 100)
}

func buildWeeklyActivity(jobs []domain.PipelineJob) []WeeklyPoint {
	respondedStages := map[domain.PipelineJobStage]bool{
		domain.StageInterview: true,
		domain.StageOffer:     true,
		domain.StageRejected:  true,
	}

	now := time.Now().UTC()
	points := make([]WeeklyPoint, 4)
	for i := 0; i < 4; i++ {
		weekOffset := 3 - i // oldest first
		monday := isoWeekMonday(now.AddDate(0, 0, -7*weekOffset))
		nextMonday := monday.AddDate(0, 0, 7)
		label := monday.Format("Jan 2")

		appCount := 0
		respCount := 0
		for _, j := range jobs {
			created := j.CreatedAt.UTC()
			if !created.Before(monday) && created.Before(nextMonday) {
				appCount++
				if respondedStages[j.Stage] && !j.IsGhosted {
					respCount++
				}
			}
		}
		points[i] = WeeklyPoint{
			WeekLabel:        label,
			ApplicationCount: appCount,
			ResponseCount:    respCount,
		}
	}
	return points
}

func computeInsight(
	buckets []ScoreRangeBucket,
	responseRate float64,
	total int,
	best *BestResume,
	thisWeekCount int,
) string {
	// Rule 1: "<60" bucket has count > 0
	for _, b := range buckets {
		if b.Label == "<60" && b.Count > 0 {
			return fmt.Sprintf(
				"Applications with ATS score below 60 have very low response rates. "+
					"You have %d pending app(s) in that range — optimize them first.",
				b.Count,
			)
		}
	}

	// Rule 2: response rate < 20 and at least 5 applications
	if responseRate < 20 && total >= 5 {
		return "Your response rate is below average. Focus on roles where your ATS score is above 80 for better results."
	}

	// Rule 3: best resume with response rate > 50
	if best != nil && best.ResponseRate > 50 {
		return fmt.Sprintf(
			"Resume version '%s' is getting a %.1f%% response rate. Use it as your default for new applications.",
			best.ResumeName,
			best.ResponseRate,
		)
	}

	// Rule 4: no applications this week
	if thisWeekCount == 0 {
		return "You haven't applied to any jobs this week. Consistent applications are key to shortening your job search."
	}

	// Rule 5: default
	return defaultInsight()
}

func defaultInsight() string {
	return "Keep applying! Candidates who apply to 10+ jobs per week are 3x more likely to land an offer within 60 days."
}
