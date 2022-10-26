// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/Masterminds/sprig"

	"github.com/saferwall/saferwall-cli/internal/entity"
	"github.com/saferwall/saferwall-cli/internal/util"
	"github.com/saferwall/saferwall-cli/internal/webapi"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// Used for flags.
var (
	soukFlag          string
	bootstrapSoukFlag bool
	familyYamlFlag    string
)

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate malware souk markdown for the entire corpus",
	Long: `Generates markdown source code for the entire corpus of
saferwall's malware souk database`,
	Run: func(cmd *cobra.Command, args []string) {

		generateMalwareSoukDB()
	},
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new malware family to the malware souk database",
	Long: `Generates markdown source code for a new malware family for
saferwall's malware souk database`,
	Run: func(cmd *cobra.Command, args []string) {
		familyYamlPath := filepath.Join(soukFlag, familyYamlFlag)
		addFamilyToSouk(familyYamlPath)
	},
}

var soukCmd = &cobra.Command{
	Use:   "souk",
	Short: "Populate malware-souk database.",
	Long:  `Generates markdown code for saferwall's malware souk database`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	soukCmd.PersistentFlags().StringVarP(&soukFlag, "souk", "s", "./",
		"Points to the malware-souk git repo (default: .current dir)")

	addCmd.Flags().StringVarP(&familyYamlFlag, "familyPath", "f", "",
		"Points to a YAML file that describes the family information")
	addCmd.MarkFlagRequired("familyPath")

	genCmd.Flags().BoolVarP(&bootstrapSoukFlag, "bootstrap", "b", false,
		"Bootstrap the malware souk database layout (default: false)")

	soukCmd.AddCommand(genCmd)
	soukCmd.AddCommand(addCmd)
}

func addFamilyToSouk(familyYamlPath string) error {
	log.Printf("processing %s", familyYamlPath)

	familyData, err := ioutil.ReadFile(familyYamlPath)
	if err != nil {
		log.Fatalf("failed to read yaml file, err: %v ", err)
	}

	var family entity.Family
	err = yaml.Unmarshal(familyData, &family)
	if err != nil {
		log.Fatalf("failed to unmarshal yaml string: %v", err)
	}

	corpusFamily := filepath.Join(soukFlag, "corpus", family.Name)
	if !util.MkDir(corpusFamily) {
		log.Fatalf("failed to create dir: %v", err)
	}

	files := map[string]entity.File{}
	for _, sample := range family.Samples {
		var file entity.File

		log.Printf("processing %s | %s | %s | %s",
			sample.SHA256, sample.Platform, sample.FileFormat, sample.Category)

		err = webapi.GetFile(sample.SHA256, &file)
		if err != nil {
			log.Fatalf("failed to read doc from saferwall web service: %v", err)
		}

		files[sample.SHA256] = file
	}

	// generate markdown for corpus.
	err = generateCorpusMarkdown(family, files)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func generateMalwareSoukDB() error {

	// Bootstrap the malware souk project file structure.
	if bootstrapSoukFlag {
		initMalwareSouk()
	}

	yamlCorpus, err := loadCorpus(soukFlag)
	if err != nil {
		log.Fatalf("failed to load corpus, err: %v ", err)
	}

	for _, yamlFamily := range yamlCorpus {
		log.Printf("processing %s", yamlFamily)

		familyData, err := ioutil.ReadFile(yamlFamily)
		if err != nil {
			log.Fatalf("failed to read yaml file, err: %v ", err)
		}

		var family entity.Family
		err = yaml.Unmarshal(familyData, &family)
		if err != nil {
			log.Fatalf("failed to unmarshal yaml string: %v", err)
		}

		corpusFamily := filepath.Join(soukFlag, "corpus", family.Name)
		if !util.MkDir(corpusFamily) {
			log.Fatalf("failed to create dir: %v", err)
		}

		files := map[string]entity.File{}
		for _, sample := range family.Samples {
			var file entity.File

			log.Printf("processing %s | %s | %s | %s",
				sample.SHA256, sample.Platform, sample.FileFormat, sample.Category)

			err = webapi.GetFile(sample.SHA256, &file)
			if err != nil {
				log.Fatalf("failed to read doc from saferwall web service: %v", err)
			}

			files[sample.SHA256] = file
		}

		// generate markdown for corpus.
		err = generateCorpusMarkdown(family, files)
		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}

func initMalwareSouk() error {
	soukYamlPath := filepath.Join(soukFlag, "souk.yaml")
	soukYamlCfg, err := ioutil.ReadFile(soukYamlPath)
	if err != nil {
		log.Printf("failed to read souk yaml file, err: %v ", err)
		return err
	}

	var m map[string]interface{}
	err = yaml.Unmarshal(soukYamlCfg, &m)
	if err != nil {
		log.Printf("failed to unmarshal yaml string: %v", err)
		return err
	}

	for k, v := range m["criteria"].(map[interface{}]interface{}) {
		criteriaName := k.(string)
		criteriaDirName := filepath.Join(soukFlag, criteriaName)
		os.Remove(criteriaDirName)
		if !util.MkDir(criteriaDirName) {
			return err
		}

		// family does not have sub criteria.
		if _, ok := v.([]interface{}); !ok {
			// drop the README.md
			filename := filepath.Join(criteriaDirName, "README.md")
			data := fmt.Sprintf("# Browse Corpus by %s:", criteriaName)
			r := bytes.NewBuffer([]byte(data))
			_, err = util.WriteBytesFile(filename, r)
			if err != nil {
				return err
			}
			continue
		}

		for _, c := range v.([]interface{}) {
			subCriteriaName := c.(string)
			subCriteriaDirName := filepath.Join(criteriaDirName, subCriteriaName)
			if !util.MkDir(subCriteriaDirName) {
				return err
			}

			// drop the README.md
			filename := filepath.Join(subCriteriaDirName, "README.md")
			data := fmt.Sprintf("# Browser Corpus by %s / %s:", criteriaName, subCriteriaName)
			r := bytes.NewBuffer([]byte(data))
			_, err = util.WriteBytesFile(filename, r)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func loadCorpus(soukFlag string) ([]string, error) {

	yamlCorpus := []string{}
	soukYamlPath := filepath.Join(soukFlag, "yaml")
	err := filepath.Walk(soukYamlPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			yamlCorpus = append(yamlCorpus, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return yamlCorpus, nil
}

func generateCorpusMarkdown(fam entity.Family, files map[string]entity.File) error {
	body := new(bytes.Buffer)

	// render the markdown
	famTemplate := filepath.Join("./templates", "family.md")

	tpl := template.Must(
		template.New("family.md").Funcs(sprig.FuncMap()).ParseFiles(famTemplate))

	data := struct {
		Fam   entity.Family
		Files map[string]entity.File
	}{
		fam,
		files,
	}

	if err := tpl.Execute(body, data); err != nil {
		return err
	}

	// create target family directory.
	corpusFamilyPath := filepath.Join(soukFlag, "corpus", fam.Name)
	if !util.Exists(corpusFamilyPath) {
		err := os.Mkdir(corpusFamilyPath, 0755)
		if err != nil {
			return err
		}
	}

	// write the family README.
	corpusFamilyReadme := filepath.Join(corpusFamilyPath, "README.md")
	_, err := util.WriteBytesFile(corpusFamilyReadme, body)
	if err != nil {
		return err
	}

	return nil
}

func generateCategoryMarkdown(fam entity.Family, files map[string]entity.File) error {
	body := new(bytes.Buffer)

	// render the markdown
	famTemplate := filepath.Join("./templates", "symlink.md")

	tpl := template.Must(
		template.New("symlink.md").Funcs(sprig.FuncMap()).ParseFiles(famTemplate))

	data := struct {
		Fam   entity.Family
		Files map[string]entity.File
	}{
		fam,
		files,
	}

	if err := tpl.Execute(body, data); err != nil {
		return err
	}

	// create target family directory.
	corpusFamilyPath := filepath.Join(soukFlag, "corpus", fam.Name)
	if !util.Exists(corpusFamilyPath) {
		err := os.Mkdir(corpusFamilyPath, 0755)
		if err != nil {
			return err
		}
	}

	// write the family README.
	corpusFamilyReadme := filepath.Join(corpusFamilyPath, "README.md")
	_, err := util.WriteBytesFile(corpusFamilyReadme, body)
	if err != nil {
		return err
	}

	return nil
}
