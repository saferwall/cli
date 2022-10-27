// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

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

	familyData, err := util.ReadAll(familyYamlPath)
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
	m := make(map[string]map[string]bool)

	m["category"] = make(map[string]bool)
	m["fileformat"] = make(map[string]bool)
	m["platform"] = make(map[string]bool)

	for _, sample := range family.Samples {
		var file entity.File

		log.Printf("processing %s | %s | %s | %s",
			sample.SHA256, sample.Platform, sample.FileFormat, sample.Category)

		err = webapi.GetFile(sample.SHA256, &file)
		if err != nil {
			log.Fatalf("failed to read doc from saferwall web service: %v", err)
		}

		files[sample.SHA256] = file

		if _, ok := m["category"][sample.Category]; !ok {
			m["category"][sample.Category] = false
		}
		if _, ok := m["platform"][sample.Platform]; !ok {
			m["platform"][sample.Platform] = false
		}
		if _, ok := m["fileformat"][sample.FileFormat]; !ok {
			m["fileformat"][sample.FileFormat] = false
		}

		// Update each criteria to link against the corpus directory.
		if !m["category"][sample.Category] {
			err = generateLink("category", sample.Category, family.Name)
			if err != nil {
				log.Fatalf("failed to generate criteria link: %v", err)
			}
		}

		if !m["platform"][sample.Platform] {
			err = generateLink("platform", sample.Platform, family.Name)
			if err != nil {
				log.Fatalf("failed to generate criteria link: %v", err)

			}
		}

		if !m["fileformat"][sample.FileFormat] {
			err = generateLink("fileformat", sample.FileFormat, family.Name)
			if err != nil {
				log.Fatalf("failed to generate criteria link: %v", err)

			}
		}

		m["category"][sample.Category] = true
		m["platform"][sample.Platform] = true
		m["fileformat"][sample.FileFormat] = true
	}

	// Generate family markdown in corpus.
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

		familyData, err := util.ReadAll(yamlFamily)
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
	soukYamlCfg, err := util.ReadAll(soukYamlPath)
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
		os.RemoveAll(criteriaDirName)
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
			data := fmt.Sprintf("# Browse corpus by %s / %s:\n\n", criteriaName, subCriteriaName)
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

func generateLink(criteria, subCriteria, familyName string) error {

	path := filepath.Join(soukFlag, criteria, subCriteria, "README.md")
	readmeData, err := util.ReadAll(path)
	if err != nil {
		return err
	}

	r := regexp.MustCompile(`# Browse corpus by .*\n\n([.\s\S]+)`)
	match := r.FindStringSubmatch(string(readmeData))

	var entries []string
	entry := fmt.Sprintf("- [%s](/corpus/%s/)", familyName, familyName)

	if len(match) == 0 {
		// This is the first time we are filling this file.
		entries = append(entries, entry)
	} else {
		entries = strings.Split(match[1], "\n")
		entries = append(entries, entry)
	}

	entries = util.UniqueSlice(entries)
	sort.Strings(entries)

	// Generate the new content.
	newContent := ""
	for _, entry := range entries {
		if entry != "" {
			newContent += entry + "\n"
		}
	}

	newReadmeData := ""
	if len(match) == 0 {
		// This is the first time we are filling this file.
		newReadmeData = string(readmeData) + newContent
	} else {
		newReadmeData = strings.ReplaceAll(string(readmeData), match[1], newContent)
	}

	_, err = util.WriteBytesFile(path, bytes.NewBufferString(newReadmeData))
	if err != nil {
		return err
	}

	return nil
}
