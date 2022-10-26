// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
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
	souk string
)

func init() {
	genCmd.Flags().StringVarP(&souk, "souk", "s", "../malware-souk",
		"Points to the malware-souk git repo (default: ../malware-souk)")
}

func loadCorpus(filename string) {
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("failed to read corpus yaml file :%v ", err)
	}

	var corpus entity.Corpus
	err = yaml.Unmarshal(yamlFile, &corpus)
	if err != nil {
		log.Fatalf("failed to unmarshal yaml string: %v", err)
	}

	for _, fam := range corpus.Families {
		log.Printf("processing %s", fam.Name)

		corpusFamily := filepath.Join(souk, "corpus", fam.Name)
		if !util.Exists(corpusFamily) {
			err = os.Mkdir(corpusFamily, 0755)
			if err != nil {
				log.Fatal(err)
			}
		}

		files := map[string]entity.File{}
		for _, sample := range fam.Samples {
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
		err = generateCorpusMarkdown(fam, files)
		if err != nil {
			log.Fatal(err)
		}
	}
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
	corpusFamilyPath := filepath.Join(souk, "corpus", fam.Name)
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
	corpusFamilyPath := filepath.Join(souk, "corpus", fam.Name)
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


var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate malware souk markdown for the entire corpus",
	Long: `Generates markdown source code for the entire corpus of
saferwall's malware souk database`,
	Run: func(cmd *cobra.Command, args []string) {

		corpusYaml := filepath.Join(souk, "corpus.yaml")
		loadCorpus(corpusYaml)
	},
}
