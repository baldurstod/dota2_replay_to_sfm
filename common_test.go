package main_test

import (
	"log"

	"github.com/baldurstod/go-source2-tools/repository"
	"github.com/baldurstod/go-source2-tools/vpk"
)

func initRepo() bool {
	repository.AddRepository("dota2", vpk.NewVpkFS("R:\\SteamLibrary\\steamapps\\common\\dota 2 beta\\game\\dota\\pak01_dir.vpk"))
	return true
}

func initLogs() bool {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	return true
}

var _ = initRepo()
var _ = initLogs()

const varFolder = "./var/"
