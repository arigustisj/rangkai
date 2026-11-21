package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arigustisj/rangkai/internal/pack"
	"github.com/spf13/cobra"
)

var (
	pushHub        string
	pushProfile    string
	pushSecrets    string
	pushSSOT       string
	pushPassphrase string
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Encrypt and upload the personal environment to a private hub over SSH",
	RunE: func(cmd *cobra.Command, args []string) error {
		pass := pushPassphrase
		if pass == "" {
			pass = os.Getenv("RANGKAI_PASSPHRASE")
		}
		if pass == "" {
			return fmt.Errorf("passphrase required (--passphrase or RANGKAI_PASSPHRASE)")
		}
		if pushHub == "" {
			return fmt.Errorf("hub required (--hub host:path)")
		}
		home, _ := os.UserHomeDir()
		secrets := pushSecrets
		if secrets == "" {
			secrets = filepath.Join(home, "secrets-central")
		}
		ssot := pushSSOT
		if ssot == "" {
			ssot = filepath.Join(home, "Dev", "00-system")
		}
		src := pack.PersonalSource{Profile: pushProfile, HermesHome: home, SecretsDir: secrets, SSOTDir: ssot}
		if err := pack.PushToHub(src, pass, pushHub, pushProfile); err != nil {
			return err
		}
		fmt.Printf("Pushed encrypted archive to %s/%s.tar.gz.age\n", pushHub, pushProfile)
		return nil
	},
}

var (
	pullHub        string
	pullProfile    string
	pullPassphrase string
	pullForce      bool
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Download and restore the personal environment from a private hub",
	RunE: func(cmd *cobra.Command, args []string) error {
		pass := pullPassphrase
		if pass == "" {
			pass = os.Getenv("RANGKAI_PASSPHRASE")
		}
		if pass == "" {
			return fmt.Errorf("passphrase required (--passphrase or RANGKAI_PASSPHRASE)")
		}
		if pullHub == "" {
			return fmt.Errorf("hub required (--hub host:path)")
		}
		home, _ := os.UserHomeDir()
		if err := pack.PullFromHub(pullHub, pullProfile, pass, home, pullForce); err != nil {
			return err
		}
		fmt.Println("Environment restored from hub.")
		return nil
	},
}

func init() {
	pushCmd.Flags().StringVar(&pushHub, "hub", "", "Hub target (ssh-host:path)")
	pushCmd.Flags().StringVarP(&pushProfile, "profile", "p", "personal", "Profile name")
	pushCmd.Flags().StringVar(&pushSecrets, "secrets", "", "Secrets directory (default $HOME/secrets-central)")
	pushCmd.Flags().StringVar(&pushSSOT, "ssot", "", "SSOT directory (default $HOME/Dev/00-system)")
	pushCmd.Flags().StringVar(&pushPassphrase, "passphrase", "", "Encryption passphrase (or RANGKAI_PASSPHRASE)")
	pullCmd.Flags().StringVar(&pullHub, "hub", "", "Hub target (ssh-host:path)")
	pullCmd.Flags().StringVarP(&pullProfile, "profile", "p", "personal", "Profile name")
	pullCmd.Flags().StringVar(&pullPassphrase, "passphrase", "", "Decryption passphrase (or RANGKAI_PASSPHRASE)")
	pullCmd.Flags().BoolVar(&pullForce, "force", false, "Overwrite an existing profile")
}
