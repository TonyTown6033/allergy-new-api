package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type cleanupCandidate struct {
	UserID       int
	Username     string
	Email        string
	SessionCount int64
	CodeCount    int64
	OrderCount   int64
}

type cleanupPlan struct {
	Candidates        []cleanupCandidate
	SkippedAdmins     []cleanupCandidate
	SkippedWithOrders []cleanupCandidate
	TotalSessionCount int64
	TotalCodeCount    int64
	TotalOrderCount   int64
}

func main() {
	execute := flag.Bool("execute", false, "apply deletions; default is dry-run")
	sampleLimit := flag.Int("sample-limit", 10, "how many sample accounts to print")
	flag.Parse()

	if err := initCleanupResources(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize cleanup resources: %v\n", err)
		os.Exit(1)
	}

	plan, err := buildCleanupPlan()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build cleanup plan: %v\n", err)
		os.Exit(1)
	}

	printCleanupPlan(plan, *sampleLimit)

	if !*execute {
		fmt.Println("mode: dry-run")
		return
	}

	if err := executeCleanupPlan(plan); err != nil {
		fmt.Fprintf(os.Stderr, "cleanup failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("cleanup complete: deleted %d users, %d sessions, %d code records\n",
		len(plan.Candidates),
		plan.TotalSessionCount,
		plan.TotalCodeCount,
	)
}

func initCleanupResources() error {
	_ = godotenv.Load(".env")
	common.InitEnv()
	logger.SetupLogger()
	return model.InitDB()
}

func buildCleanupPlan() (*cleanupPlan, error) {
	var profiles []model.MemberProfile
	if err := model.DB.Where("email_verified_at IS NULL").Find(&profiles).Error; err != nil {
		return nil, err
	}

	plan := &cleanupPlan{
		Candidates:        make([]cleanupCandidate, 0, len(profiles)),
		SkippedAdmins:     make([]cleanupCandidate, 0),
		SkippedWithOrders: make([]cleanupCandidate, 0),
	}

	hasAllergyOrdersTable := model.DB.Migrator().HasTable(&model.AllergyOrder{})

	for _, profile := range profiles {
		user, err := model.GetUserById(profile.UserID, false)
		if err != nil {
			continue
		}

		candidate := cleanupCandidate{
			UserID:   user.Id,
			Username: user.Username,
			Email:    model.NormalizeEmail(user.Email),
		}

		if err := model.DB.Model(&model.MemberSession{}).Where("user_id = ?", user.Id).Count(&candidate.SessionCount).Error; err != nil {
			return nil, err
		}
		if candidate.Email != "" {
			if err := model.DB.Model(&model.EmailLoginCodeStore{}).Where("email = ?", candidate.Email).Count(&candidate.CodeCount).Error; err != nil {
				return nil, err
			}
		}
		if hasAllergyOrdersTable {
			if err := model.DB.Model(&model.AllergyOrder{}).Where("user_id = ?", user.Id).Count(&candidate.OrderCount).Error; err != nil {
				return nil, err
			}
		}

		if user.Role >= common.RoleAdminUser {
			plan.SkippedAdmins = append(plan.SkippedAdmins, candidate)
			continue
		}
		if candidate.OrderCount > 0 {
			plan.SkippedWithOrders = append(plan.SkippedWithOrders, candidate)
			continue
		}

		plan.Candidates = append(plan.Candidates, candidate)
		plan.TotalSessionCount += candidate.SessionCount
		plan.TotalCodeCount += candidate.CodeCount
		plan.TotalOrderCount += candidate.OrderCount
	}

	sort.Slice(plan.Candidates, func(i, j int) bool {
		return plan.Candidates[i].UserID < plan.Candidates[j].UserID
	})

	return plan, nil
}

func executeCleanupPlan(plan *cleanupPlan) error {
	if len(plan.Candidates) == 0 {
		return nil
	}

	userIDs := make([]int, 0, len(plan.Candidates))
	emails := make([]string, 0, len(plan.Candidates))
	for _, candidate := range plan.Candidates {
		userIDs = append(userIDs, candidate.UserID)
		if candidate.Email != "" {
			emails = append(emails, candidate.Email)
		}
	}

	return model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id IN ?", userIDs).Delete(&model.MemberSession{}).Error; err != nil {
			return err
		}
		if len(emails) > 0 {
			if err := tx.Where("email IN ?", emails).Delete(&model.EmailLoginCodeStore{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("user_id IN ?", userIDs).Delete(&model.MemberProfile{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("id IN ?", userIDs).Delete(&model.User{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func printCleanupPlan(plan *cleanupPlan, sampleLimit int) {
	fmt.Printf("deletable legacy members: %d\n", len(plan.Candidates))
	fmt.Printf("affected member_session rows: %d\n", plan.TotalSessionCount)
	fmt.Printf("affected email_login_code_store rows: %d\n", plan.TotalCodeCount)
	fmt.Printf("protected admins: %d\n", len(plan.SkippedAdmins))
	fmt.Printf("skipped users with allergy orders: %d\n", len(plan.SkippedWithOrders))

	if sampleLimit <= 0 {
		return
	}

	if len(plan.Candidates) > 0 {
		fmt.Println("sample deletable accounts:")
		for _, candidate := range plan.Candidates[:min(sampleLimit, len(plan.Candidates))] {
			fmt.Printf("  - #%d %s <%s> sessions=%d codes=%d\n",
				candidate.UserID,
				candidate.Username,
				maskEmail(candidate.Email),
				candidate.SessionCount,
				candidate.CodeCount,
			)
		}
	}

	if len(plan.SkippedWithOrders) > 0 {
		fmt.Println("sample skipped accounts with allergy orders:")
		for _, candidate := range plan.SkippedWithOrders[:min(sampleLimit, len(plan.SkippedWithOrders))] {
			fmt.Printf("  - #%d %s <%s> orders=%d\n",
				candidate.UserID,
				candidate.Username,
				maskEmail(candidate.Email),
				candidate.OrderCount,
			)
		}
	}
}

func maskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	local := parts[0]
	if len(local) <= 2 {
		return local[:1] + "***@" + parts[1]
	}
	return local[:2] + "***@" + parts[1]
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
