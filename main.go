package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type NasmConfig struct {
	NasmPath string
	Gcc32Path string
	Gcc64Path string
}

var (
	nasmConfig NasmConfig
	source string
	mode string //win32, win64
	wd string
)

const config = "config.toml"

func main() {
	flag.StringVar(&source, "source", "", "source file path")
	flag.StringVar(&mode, "mode", "", "mode")
	flag.Parse()

	if len(mode) == 0 || len(source) == 0 {
		log.Fatal("Please specify both --mode and --source arguments")
	}

	wd, _ = os.Getwd()

	if _, err := toml.DecodeFile(filepath.Join(appPath(), config), &nasmConfig); err != nil {
		if e, ok := err.(*os.PathError); ok && e.Err  == syscall.ENOENT {
			var buffer bytes.Buffer
			e := toml.NewEncoder(&buffer)

			nasmConfig = NasmConfig{
				NasmPath: "",
				Gcc32Path: "",
				Gcc64Path: "",
			}

			err := e.Encode(nasmConfig)
			err = ioutil.WriteFile(filepath.Join(appPath(), config), buffer.Bytes(), 0644)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("Please set up config.toml before use.")
		} else {
			fmt.Println("Cannot parse content correctly, did you escape path slash?")
		}
		return
	}

	err := os.Chdir(filepath.Join(appPath(), "include"))
	if err != nil{
		fmt.Println(err)
	}

	fmt.Println("NASM Path:", nasmConfig.NasmPath)
	fmt.Println("GCC 32 Path:", nasmConfig.Gcc32Path)
	fmt.Println("GCC 64 Path:", nasmConfig.Gcc64Path)

	Compile()
	Link()
}

func Compile() {
	log.Println("Compiling")
	var args = []string{"-g", "-f"}
	checkMode(func() {
		args = append(args, "win32")
	}, func() {
		args = append(args, "win64")
	})
	Command(nasmConfig.NasmPath, append(args, filepath.Join(wd, source), "-l", filepath.Join(wd, fmt.Sprintf("%s.lst", removeExtension(source))), "-o", filepath.Join(wd, fmt.Sprintf("%s.o", removeExtension(source))))...)
	log.Println("Compile complete")
}

func Link() {
	log.Println("Linking")
	var linkerPath = ""
	var marcoPath = ""
	var linkerSwitch = ""
	checkMode(func() {
		linkerPath = nasmConfig.Gcc32Path
		marcoPath = filepath.Join(appPath(), "macro\\macro.o")
		linkerSwitch = "-m32"
	}, func() {
		linkerPath = nasmConfig.Gcc64Path
		marcoPath = filepath.Join(appPath(), "macro\\macro64.o")
		linkerSwitch = "-m64"
	})

	Command(linkerPath, append(
		append([]string{filepath.Join(wd, fmt.Sprintf("%s.o", removeExtension(source)))}, marcoPath),
		"-g", "-o", filepath.Join(wd, removeExtension(source)), linkerSwitch)...)

	log.Println("Link complete")
}

func Command(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	stdErr, _ := cmd.StderrPipe()
	cmd.Start()
	errBytes, _ := ioutil.ReadAll(stdErr)
	cmd.Wait()
	err := string(errBytes)
	if len(err) > 0 {
		panic(err)
	}
}

func removeExtension(filename string) string {
	return filename[0:len(filename)-len(filepath.Ext(filename))]
}

func checkMode(win32Callback, win64Callback func()) {
	if mode == "win32" {
		win32Callback()
	} else if mode == "win64" {
		win64Callback()
	}
}

func appPath() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(ex)
}