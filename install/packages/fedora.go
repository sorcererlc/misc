package packages

import (
	"errors"
	"fmt"
	"os"
	"setup/helper"
	"setup/log"
	"setup/types"
	"strings"

	"gopkg.in/yaml.v3"
)

type FedoraHelper struct {
	Conf      *types.Config
	Env       *types.Environment
	Log       *log.Log
	CoprRepos struct {
		Copr []string `yaml:"copr"`
	}
}

func NewFedoraHelper(c *types.Config, e *types.Environment) *FedoraHelper {
	f := FedoraHelper{
		Conf: c,
		Env:  e,
		Log:  log.NewLog("packages.log"),
	}

	return &f
}

func (f *FedoraHelper) SetupPackages(pkg *types.Packages) error {
	err := f.updateDistro()
	if err != nil {
		return err
	}

	err = f.enableCoprRepos()
	if err != nil {
		return err
	}

	err = f.removePackages(pkg.Remove)
	if err != nil {
		return err
	}

	err = f.installRepos(pkg.Repo)
	if err != nil {
		return err
	}

	p := pkg.Base
	if f.Conf.Packages.Extras {
		p = append(p, pkg.Extras...)
	}
	if f.Conf.Packages.Sddm {
		p = append(p, pkg.Sddm...)
	}
	if f.Conf.Packages.Bluetooth {
		p = append(p, pkg.Bluetooth...)
	}
	if f.Conf.Packages.Nvidia {
		p = append(p, pkg.Nvidia...)
	}
	p = append(p, pkg.Fonts...)

	err = f.installPackages(p)
	if err != nil {
		return err
	}

	err = f.installAdvCpMv()
	if err != nil {
		return err
	}

	err = f.installAutoCpuFreq(pkg.Git["auto-cpufreq"])
	if err != nil {
		return err
	}

	err = f.setupNwgLook(pkg.Git["nwg-look"])
	if err != nil {
		return err
	}

	return nil
}

func (f *FedoraHelper) updateDistro() error {
	f.Log.Info("Updating packages")

	err := helper.Run("sudo", "dnf", "upgrade", "-y")
	if err != nil {
		f.Log.Error("Update packages", err.Error())
		return err
	}

	return nil
}

func (f *FedoraHelper) installRepos(r []string) error {
	for i := 0; i < len(r); i++ {
		r[i] = fmt.Sprintf(r[i], f.Env.OS.VersionId)
	}
	f.Log.Info("Installing repositories", strings.Join(r, ", "))

	args := []string{"sudo", "dnf", "install", "-y"}
	args = append(args, r...)
	err := helper.Run(args...)
	if err != nil {
		f.Log.Error("Install repositories", err.Error())
		return err
	}

	return nil
}

func (f *FedoraHelper) enableCoprRepos() error {
	f.Log.Info("Enabling Copr repositories", strings.Join(f.CoprRepos.Copr, ", "))

	fs, err := os.ReadFile(f.Env.Cwd + "/packages/fedora/repos.yml")
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(fs, &f.CoprRepos)

	args := []string{"sudo", "dnf", "copr", "enable", "-y"}
	args = append(args, f.CoprRepos.Copr...)
	err = helper.Run(args...)
	if err != nil {
		f.Log.Error("Enable Copr repositories", err.Error())
		return err
	}

	return nil
}

func (f *FedoraHelper) removePackages(p []string) error {
	f.Log.Info("Removing packages", strings.Join(p, ", "))

	args := []string{"sudo", "dnf", "remove", "-y"}
	args = append(args, p...)
	err := helper.Run(args...)
	if err != nil {
		f.Log.Error("Remove packages", err.Error())
		return err
	}

	return nil
}

func (f *FedoraHelper) installPackages(p []string) error {
	f.Log.Info("Installing packages", strings.Join(p, ", "))

	args := []string{"sudo", "dnf", "install", "-y"}
	args = append(args, p...)
	err := helper.Run(args...)
	if err != nil {
		f.Log.Error("Install packages", err.Error())
		return err
	}

	for _, pk := range p {
		i := f.checkInstalledPackage(pk)
		if !i {
			f.Log.Error("Package " + pk + " failed to install. Aborting setup.")
			return errors.New("")
		}
	}

	return nil
}

func (f *FedoraHelper) checkInstalledPackage(p string) bool {
	err := helper.Run("dnf", "list", "installed", p)
	if err != nil {
		return false
	}

	return true
}

func (f *FedoraHelper) runGitCommand(c string) error {
	args := strings.Split(c, " ")

	return helper.Run(args...)
}

func (f *FedoraHelper) setupNwgLook(p types.GitPackage) error {
	f.Log.Info("Installing nwg-look")

	err := helper.Run("git", "clone", "--recursive", "--depth", "1", "--branch", p.Tag, p.Url)
	if err != nil {
		f.Log.Error("Clone nwg-look repo", err.Error())
		return err
	}

	for _, c := range p.Commands {
		err := f.runGitCommand(c)
		if err != nil {
			f.Log.Error(c, err.Error())
			return err
		}
	}

	return nil
}

func (f *FedoraHelper) installAutoCpuFreq(p types.GitPackage) error {
	f.Log.Info("Installing auto-cpufreq")

	err := helper.Run("git", "clone", "--recursive", "--depth", "1", p.Url)
	if err != nil {
		f.Log.Error("Clone auto-cpufreq repo", err.Error())
		return err
	}

	for _, c := range p.Commands {
		err := f.runGitCommand(c)
		if err != nil {
			f.Log.Error(c, err.Error())
			return err
		}
	}

	return nil
}

func (f *FedoraHelper) installAdvCpMv() error {
	f.Log.Info("Installing advcpmv")

	err := helper.Run("curl", "https://raw.githubusercontent.com/jarun/advcpmv/master/install.sh", "--create-dirs", "-o", "./advcpmv/install.sh", "&&", "(cd", "advcpmv", "&&", "sh", "install.sh)")
	if err != nil {
		f.Log.Error("Install advcpmv", err.Error())
		return err
	}

	err = helper.Run("sudo", "mv", "./advcpmv/advcp", "/usr/local/bin/cpg")
	err = helper.Run("sudo", "mv", "./advcpmv/advmv", "/usr/local/bin/mvg")
	if err != nil {
		f.Log.Error("Move advcpmv binaries", err.Error())
		return err
	}

	err = helper.Run("rm", "-rf", "./advcpmv")
	if err != nil {
		f.Log.Error("Cleanup advcpmv", err.Error())
		return err
	}

	return nil
}
