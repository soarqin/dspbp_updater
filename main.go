package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/pterm/pterm"
)

const (
	REMOTE_NAME = "origin"
	BRANCH_NAME = "main"
	REPO_URL    = "https://github.com/DSPBluePrints/FactoryBluePrints.git"
)

var (
	warnStyle = pterm.NewStyle(pterm.BgBlack, pterm.FgLightRed, pterm.Bold)
	infoStyle = pterm.NewStyle(pterm.BgBlue, pterm.FgLightWhite)
	// QuestionStyle = pterm.NewStyle(pterm.BgDarkGray, pterm.FgYellow, pterm.Bold)
)

func waitForKeyInput() {
	infoStyle.Println("Press any key to exit...")
	bufio.NewReader(os.Stdin).ReadByte()
}

func checkError(err error, msg string) {
	if err != nil {
		warnStyle.Println(fmt.Sprintf("%s: %v", msg, err))
		waitForKeyInput()
		os.Exit(1)
	}
}

func main() {
	repo, err := git.PlainOpen(".")
	repoAvailable := err == nil
	if !repoAvailable {
		if _, err := os.Stat(".git"); err == nil {
			if err := os.RemoveAll(".git"); err != nil {
				checkError(err, "Failed to remove .git directory")
				os.Exit(1)
			}
		}
		repo, err = git.PlainInit(".", false)
		checkError(err, "Failed to initialize repository")
	}
	needReopen := false
	for {
		var option1 string
		if repoAvailable {
			option1 = "Update Repository"
		} else {
			option1 = "Clone Repository"
		}
		printer := pterm.DefaultInteractiveSelect.
			WithOptions([]string{option1, "Set mirror (for China Mainland users)"}).
			WithFilter(false).
			WithDefaultText("Choose an operation")
		selected, err := printer.Show()
		checkError(err, "Failed to show interactive select")
		if selected == option1 {
			break
		}
		printer2 := pterm.DefaultInteractiveSelect.
			WithOptions([]string{"GitHub (no mirror)", "Codeberg"}).
			WithFilter(false).
			WithDefaultText("Select a mirror")
		selected, err = printer2.Show()
		checkError(err, "Failed to show interactive select")
		switch selected {
		case "GitHub (no mirror)":
			cfg, err := repo.Config()
			checkError(err, "Failed to get config")
			delete(cfg.URLs, "https://codeberg.org/")
			err = repo.SetConfig(cfg)
			checkError(err, "Failed to set config")
			needReopen = true
		case "Codeberg":
			cfg, err := repo.Config()
			checkError(err, "Failed to get config")
			cfg.URLs["https://codeberg.org/"] = &config.URL{
				Name:       "https://codeberg.org/",
				InsteadOfs: []string{"https://github.com/"},
			}
			err = repo.SetConfig(cfg)
			checkError(err, "Failed to set config")
			needReopen = true
		}
	}
	if !repoAvailable {
		infoStyle.Println("Starting to clone repository...")
	} else {
		infoStyle.Println("Starting to update repository...")
	}
	remote, err := repo.Remote(REMOTE_NAME)
	if err != nil {
		remote, err = repo.CreateRemote(&config.RemoteConfig{
			Name: REMOTE_NAME,
			URLs: []string{REPO_URL},
		})
		checkError(err, "Failed to create remote")
		cfg, err := repo.Config()
		checkError(err, "Failed to get config")
		remoteConfig := cfg.Remotes[REMOTE_NAME]
		found := len(remoteConfig.URLs) > 0 && remoteConfig.URLs[0] == REPO_URL
		if !found {
			if len(remoteConfig.URLs) == 0 {
				remoteConfig.URLs = append(remoteConfig.URLs, REPO_URL)
			} else {
				remoteConfig.URLs[0] = REPO_URL
			}
			err = repo.SetConfig(cfg)
			checkError(err, "Failed to set config")
		}
		needReopen = true
	}
	if needReopen {
		// Reopen repository to apply insteadOf fields to remote URLs
		repo, err = git.PlainOpen(".")
		checkError(err, "Failed to open repository")
		remote, err = repo.Remote(REMOTE_NAME)
		checkError(err, "Failed to get remote")
	}
	err = remote.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec("refs/heads/*:refs/remotes/" + REMOTE_NAME + "/*"),
		},
		Prune:    true,
		Progress: os.Stdout,
	})
	alreadyUpToDate := err == git.NoErrAlreadyUpToDate
	if !alreadyUpToDate {
		checkError(err, "Failed to fetch remote")
	}
	w, err := repo.Worktree()
	checkError(err, "Failed to get worktree")
	head, err := repo.Head()
	if head == nil || head.Name().Short() != BRANCH_NAME {
		if head != nil {
			infoStyle.Println(fmt.Sprintf("Repository is not on %s branch.", BRANCH_NAME))
			printer := pterm.DefaultInteractiveSelect.
				WithOptions([]string{"Yes", "No"}).
				WithFilter(false).
				WithDefaultText(fmt.Sprintf("Do you want to checkout to %s branch? (Yes/No)", BRANCH_NAME))
			selected, err := printer.Show()
			checkError(err, "Failed to show interactive select")
			if selected == "No" {
				return
			}
		}
		cfg, err := repo.Config()
		checkError(err, "Failed to get config")
		if _, ok := cfg.Branches[BRANCH_NAME]; !ok {
			remoteRef, err := repo.Reference(plumbing.NewRemoteReferenceName(REMOTE_NAME, BRANCH_NAME), true)
			checkError(err, "Failed to get remote reference")
			localBranchRef := plumbing.NewBranchReferenceName(BRANCH_NAME)
			ref := plumbing.NewHashReference(localBranchRef, remoteRef.Hash())
			err = repo.Storer.SetReference(ref)
			checkError(err, "Failed to set reference")
			err = w.Checkout(&git.CheckoutOptions{
				Branch: localBranchRef,
			})
			checkError(err, fmt.Sprintf("Failed to checkout to %s branch", BRANCH_NAME))
			cfg.Branches[BRANCH_NAME] = &config.Branch{
				Name:   BRANCH_NAME,
				Remote: REMOTE_NAME,
				Merge:  plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", BRANCH_NAME)),
			}
			err = repo.SetConfig(cfg)
			checkError(err, "Failed to set config")
		} else {
			err = w.Checkout(&git.CheckoutOptions{
				Branch: plumbing.NewBranchReferenceName(BRANCH_NAME),
			})
			checkError(err, fmt.Sprintf("Failed to checkout to %s branch", BRANCH_NAME))
		}
		if head != nil {
			infoStyle.Println(fmt.Sprintf("Repository has been checked out to %s branch.", BRANCH_NAME))
		}
	}
	status, err := w.Status()
	checkError(err, "Failed to get status")
	if !statusIsClean(&status) {
		infoStyle.Println("Repository is not clean, please consider committing or stashing your changes first.")
		printer := pterm.DefaultInteractiveSelect.
			WithOptions([]string{"Yes", "No"}).
			WithFilter(false).
			WithDefaultText("Do you want to reset the repository anyway? (Yes/No)")
		selected, err := printer.Show()
		checkError(err, "Failed to show interactive select")
		if selected == "No" {
			return
		}
		err = w.Reset(&git.ResetOptions{
			Mode: git.HardReset,
		})
		checkError(err, "Failed to reset worktree")
		infoStyle.Println("Repository has been reset.")
	}
	if !alreadyUpToDate {
		err = w.Pull(&git.PullOptions{
			RemoteName:   REMOTE_NAME,
			Progress:     os.Stdout,
			SingleBranch: true,
		})
		if err == git.NoErrAlreadyUpToDate {
			infoStyle.Println("Repository is already up to date.")
			waitForKeyInput()
			return
		} else {
			checkError(err, "Failed to pull repository")
		}
		infoStyle.Println("Repository has been pulled and updated.")
	} else {
		infoStyle.Println("Repository is already up to date.")
	}
	waitForKeyInput()
}

func statusIsClean(s *git.Status) bool {
	for name, status := range *s {
		if name == "dspbp_updater.exe" {
			continue
		}
		if status.Worktree != git.Unmodified || status.Staging != git.Unmodified {
			return false
		}
	}
	return true
}
