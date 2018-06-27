package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// context : Store specific value to alter the program behaviour
// Like an Args container
type (
	filespec struct {
		name string
		path string
	}

	context struct {
		files   *string
		mins    uint64
		output  *string
		verbose bool
		// filel   []os.FileInfo
		filel []filespec
	}
)

// contexte : Hold runtime value (from commande line args)
var contexte context

// Check if path contains Wildcard characters
func isWildcard(value string) bool {
	return strings.Contains(value, "*") || strings.Contains(value, "?")
}

// Get the files list to copy
func getFiles(ctx *context, info string) error {
	// fmt.Println("Processing :", info)
	pattern := filepath.Base(info)
	files, err := ioutil.ReadDir(filepath.Dir(info))
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		if res, err := filepath.Match(strings.ToLower(pattern), strings.ToLower(file.Name())); res {
			if err != nil {
				return err
			}
			if file.ModTime().After(time.Now().Add(-1 * time.Minute * time.Duration(ctx.mins))) {
				ctx.filel = append(ctx.filel, filespec{name: file.Name(), path: filepath.Dir(info)})
				// fmt.Printf("prise en compte de %s\n", file.Name())
				// } else {
				// 	fmt.Printf("fichier %s trop vieux (%v)\n", file.Name(), file.ModTime())
			}
		}
	}
	// for i, file := range ctx.filel {
	// 	fmt.Printf("(%d) - %s dans %s", i, file.name, file.path)
	// }

	return nil
}

// Get the files list to copy
func exploitFiles(ctx *context) error {
	items := strings.SplitN(*ctx.files, ";", -1)
	for i, item := range items {
		err := getFiles(ctx, item)
		if err != nil {
			log.Println("Can't retrieve files with", i, item)
			return err
		}
	}
	return nil
}

func main() {
	log.Println("ziplog - Logfiles zipper (but not only) - C.m. V1.0")

	filesPtr := flag.String("files", "*.log", "File spec(s) to process (semicolon as separator)")
	minutesPtr := flag.Uint64("minutes", 60, "Last updated time in minutes")
	outputPtr := flag.String("output", "ziplog.zip", "Output file (zip format)")
	verbosePtr := flag.Bool("verbose", false, "Verbose mode")

	flag.Parse()

	contexte.files = filesPtr
	contexte.mins = *minutesPtr
	contexte.output = outputPtr
	contexte.verbose = *verbosePtr

	err := exploitFiles(&contexte)
	if err != nil {
		return
	}

	err = ZipFiles(*contexte.output, contexte.filel)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Zipped %d Files in %s", len(contexte.filel), *contexte.output)

}

// ZipFiles compresses one or many files into a single zip archive file
func ZipFiles(filename string, files []filespec) error {

	newfile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newfile.Close()

	zipWriter := zip.NewWriter(newfile)
	defer zipWriter.Close()

	// Add files to zip
	for _, file := range files {

		zipfile, err := os.OpenFile(file.path+"\\"+file.name, os.O_RDONLY, 0)
		if err != nil {
			fmt.Println("error on", file.path+"\\"+file.name, err)
			continue
		}
		defer zipfile.Close()

		// Get the file information
		info, err := zipfile.Stat()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Change to deflate to gain better compression
		// see http://golang.org/pkg/archive/zip/#pkg-constants
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, zipfile)
		if err != nil {
			return err
		}
	}
	return nil
}
