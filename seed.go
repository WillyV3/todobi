package main

import (
	"time"
)

// SeedWeekendTasks creates a sample configuration with your weekend tasks
func SeedWeekendTasks() *Config {
	now := time.Now()

	return &Config{
		Version:    "1.0.0",
		LastUpdate: now,
		Groups: []TaskGroup{
			{
				Name:     "Priority 0: Critical Revenue Blockers (Eldercare)",
				Priority: P0Critical,
				Tasks: []Task{
					{
						ID:        generateTaskID(),
						Content:   "#1 - Consultation Booking Page UI",
						Priority:  P0Critical,
						CreatedAt: now,
						URL:       "https://github.com/Human-Frontier-Labs-Inc/eldercare/issues/1",
					},
					{
						ID:        generateTaskID(),
						Content:   "#2 - Consultation Scheduling Calendar UI",
						Priority:  P0Critical,
						CreatedAt: now,
						URL:       "https://github.com/Human-Frontier-Labs-Inc/eldercare/issues/2",
					},
					{
						ID:        generateTaskID(),
						Content:   "#3 - Subscription Upgrade Modal",
						Priority:  P0Critical,
						CreatedAt: now,
						URL:       "https://github.com/Human-Frontier-Labs-Inc/eldercare/issues/3",
					},
					{
						ID:        generateTaskID(),
						Content:   "#4 - Subscription Tier Badge in Header",
						Priority:  P0Critical,
						CreatedAt: now,
						URL:       "https://github.com/Human-Frontier-Labs-Inc/eldercare/issues/4",
					},
				},
			},
			{
				Name:     "Priority 1: Core Functionality (Eldercare)",
				Priority: P1High,
				Tasks: []Task{
					{
						ID:        generateTaskID(),
						Content:   "#5 - Consultations List & History Page",
						Priority:  P1High,
						CreatedAt: now,
						URL:       "https://github.com/Human-Frontier-Labs-Inc/eldercare/issues/5",
					},
					{
						ID:        generateTaskID(),
						Content:   "#6 - Optimize Server Actions - Consultation Domain",
						Priority:  P1High,
						CreatedAt: now,
						URL:       "https://github.com/Human-Frontier-Labs-Inc/eldercare/issues/6",
					},
					{
						ID:        generateTaskID(),
						Content:   "#7 - Optimize Server Actions - Subscription Domain",
						Priority:  P1High,
						CreatedAt: now,
						URL:       "https://github.com/Human-Frontier-Labs-Inc/eldercare/issues/7",
					},
					{
						ID:        generateTaskID(),
						Content:   "#8 - Optimize Server Actions - AI Chat Domain",
						Priority:  P1High,
						CreatedAt: now,
						URL:       "https://github.com/Human-Frontier-Labs-Inc/eldercare/issues/8",
					},
					{
						ID:        generateTaskID(),
						Content:   "#9 - AI Message Quota Display in Chat UI",
						Priority:  P1High,
						CreatedAt: now,
						URL:       "https://github.com/Human-Frontier-Labs-Inc/eldercare/issues/9",
					},
				},
			},
			{
				Name:     "Homelab Infrastructure",
				Priority: PHHomelab,
				Tasks: []Task{
					{
						ID:          generateTaskID(),
						Content:     "Fix existing homelab issues",
						Description: "Diagnose and repair current homelab infrastructure problems",
						Priority:    PHHomelab,
						CreatedAt:   now,
					},
					{
						ID:          generateTaskID(),
						Content:     "Implement distributed file sharing across Tailscale network",
						Description: "Set up NFS/Samba/Syncthing for seamless file sharing",
						Priority:    PHHomelab,
						CreatedAt:   now,
					},
					{
						ID:          generateTaskID(),
						Content:     "Wire Sonia's MacBook into Tailscale network",
						Description: "Install Tailscale and configure networking on Sonia's machine",
						Priority:    PHHomelab,
						CreatedAt:   now,
					},
					{
						ID:          generateTaskID(),
						Content:     "Design multi-Claude architecture with shared filesystem",
						Description: "Enable cross-machine Claude collaboration to eliminate system constraints",
						Priority:    PHHomelab,
						CreatedAt:   now,
					},
				},
			},
			{
				Name:     "Development Environment Standardization",
				Priority: PDev,
				Tasks: []Task{
					{
						ID:          generateTaskID(),
						Content:     "Review master-claude-work repo structure",
						Description: "Understand standardization patterns in the new repo",
						Priority:    PDev,
						CreatedAt:   now,
						URL:         "https://github.com/WillyV3/master-claude-work",
					},
					{
						ID:          generateTaskID(),
						Content:     "Review gummy-agent implementation",
						Description: "Study gummy-agent for integration patterns",
						Priority:    PDev,
						CreatedAt:   now,
						URL:         "https://github.com/WillyV3/gummy-agent",
					},
					{
						ID:          generateTaskID(),
						Content:     "Standardize work/personal configurations",
						Description: "Merge best practices from both environments",
						Priority:    PDev,
						CreatedAt:   now,
					},
					{
						ID:          generateTaskID(),
						Content:     "Clean up system inconsistencies",
						Description: "Remove duplicate configs and consolidate tooling",
						Priority:    PDev,
						CreatedAt:   now,
					},
					{
						ID:          generateTaskID(),
						Content:     "Document standard setup process",
						Description: "Create reproducible setup instructions",
						Priority:    PDev,
						CreatedAt:   now,
					},
				},
			},
		},
	}
}
