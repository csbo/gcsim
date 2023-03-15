package simulator

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/genshinsim/gcsim/pkg/agg"
	"github.com/genshinsim/gcsim/pkg/gcs/ast"
	"github.com/genshinsim/gcsim/pkg/result"
	"github.com/genshinsim/gcsim/pkg/stats"
	"github.com/genshinsim/gcsim/pkg/worker"
)

// Options sets out the settings to run the sim by (such as debug mode, etc..)
type Options struct {
	ResultSaveToPath string // file name (excluding ext) to save the result file; if "" then nothing is saved to file
	GZIPResult       bool   // should the result file be gzipped; only if ResultSaveToPath is not ""
	ConfigPath       string // path to the config file to read
	Version          string
	BuildDate        string
	DebugMinMax      bool // whether to additional include debug logs for min/max-DPS runs
}

var start time.Time

// Run will run the simulation given number of times
func Run(opts Options) (result.Summary, error) {
	start = time.Now()
	//err := nil
	//cfg, err := ReadConfig(opts.ConfigPath)
	//if err != nil {
	//	return result.Summary{}, err
	//}
	cfg := `baizhu char lvl=90/90 cons=0 talent=6,9,9;
	baizhu add weapon="prototypeamber" refine=5 lvl=90/90;
	baizhu add set="oceanhuedclam" count=4;
	baizhu add stats def%=0 def=0 hp=4780 hp%=0.466 atk=311 atk%=0 er=0.518 em=0 cr=0 cd=0 heal=0.359; #main
	baizhu add stats def%=0 def=0 hp=0 hp%=0.8 atk=0 atk%=0 er=0.44 em=0 cr=0 cd=0 heal=0;
	raiden char lvl=90/90 cons=0 talent=6,9,9;
	raiden add weapon="dragonsbane" refine=5 lvl=90/90;
	raiden add set="flowerofparadiselost" count=4;
	raiden add stats def%=0.124 def=97 hp=6693 atk=325 atk%=0.23300000000000004 er=0.097 em=641.5 cr=0.19 cd=0.653;
	xingqiu char lvl=90/90 cons=6 talent=1,9,9;
	xingqiu add weapon="favoniussword" refine=5 lvl=90/90;
	xingqiu add set="deepwoodmemories" count=4;
	xingqiu add stats def%=0 hp=5087 hp%=0.053 atk=476 atk%=0.7630000000000001 er=0.16199999999999998 em=44 cr=0.66 cd=0.435 hydro%=0.466;
	yelan char lvl=90/90 cons=1 talent=2,9,9;
	yelan add weapon="slingshot" refine=5 lvl=90/90;
	yelan add set="emblemofseveredfate" count=4;
	yelan add stats def%=0.057999999999999996 def=23 hp=5378 hp%=0.932 atk=391 atk%=0.057999999999999996 er=0.498 cr=0.231 cd=1.046 hydro%=0.466;

	target lvl=100 resist=0.1 pos=0,0;
	#energy every interval=60,80 amount=10;
	options swap_delay=4 debug=true iteration=50 duration=500;
	
	active baizhu;
	baizhu burst;
	while 1 {
		raiden skill;
		xingqiu skill[orbital=0], burst[orbital=0], attack;
		if .xingqiu.skill.ready {
			xingqiu skill[orbital=0];
		}
		yelan skill, burst;
		baizhu skill;
		baizhu attack:2, dash, attack:2, dash, attack:2, dash, attack:2, dash;
		baizhu attack:2, dash, attack:2, dash, attack:2, dash, attack:2, dash;
		baizhu attack:2, skill;
		baizhu attack:2, dash, attack:2, dash, attack:2, dash;
		baizhu attack, burst;
	}
	`
	parser := ast.New(cfg)
	simcfg, err := parser.Parse()
	if err != nil {
		return result.Summary{}, err
	}
	//check other errors as well
	if len(simcfg.Errors) != 0 {
		fmt.Println("The config has the following errors: ")
		for _, v := range simcfg.Errors {
			fmt.Printf("\t%v\n", v)
		}
		return result.Summary{}, errors.New("sim has errors")
	}
	return RunWithConfig(cfg, simcfg, opts)
}

// Runs the simulation with a given parsed config
func RunWithConfig(cfg string, simcfg *ast.ActionList, opts Options) (result.Summary, error) {
	// initialize aggregators
	var aggregators []agg.Aggregator
	for _, aggregator := range agg.Aggregators() {
		a, err := aggregator(simcfg)
		if err != nil {
			return result.Summary{}, err
		}
		aggregators = append(aggregators, a)
	}

	//set up a pool
	respCh := make(chan stats.Result)
	errCh := make(chan error)
	pool := worker.New(simcfg.Settings.NumberOfWorkers, respCh, errCh)
	pool.StopCh = make(chan bool)

	//spin off a go func that will queue jobs for as long as the total queued < iter
	//this should block as queue gets full
	go func() {
		//make all the seeds
		wip := 0
		for wip < simcfg.Settings.Iterations {
			pool.QueueCh <- worker.Job{
				Cfg:  simcfg.Copy(),
				Seed: CryptoRandSeed(),
			}
			wip++
		}
	}()

	defer close(pool.StopCh)

	//start reading respCh, queueing a new job until wip == number of iterations
	count := 0
	for count < simcfg.Settings.Iterations {
		select {
		case result := <-respCh:
			for _, a := range aggregators {
				a.Add(result, count)
			}
			count += 1
		case err := <-errCh:
			//error encountered
			return result.Summary{}, err
		}
	}

	// generate final agg results
	stats := &agg.Result{}
	for _, a := range aggregators {
		a.Flush(stats)
	}

	result, err := GenerateResult(cfg, simcfg, stats, opts)
	if err != nil {
		return result, err
	}

	//TODO: clean up this code
	if opts.ResultSaveToPath != "" {
		err = result.Save(opts.ResultSaveToPath, opts.GZIPResult)
		if err != nil {
			return result, err
		}
	}

	return result, nil
}

func GenerateResult(cfg string, simcfg *ast.ActionList, stats *agg.Result, opts Options) (result.Summary, error) {
	result := result.Summary{
		V2:            true,
		Version:       opts.Version,
		BuildDate:     opts.BuildDate,
		IsDamageMode:  simcfg.Settings.DamageMode,
		ActiveChar:    simcfg.InitialChar.String(),
		Iterations:    simcfg.Settings.Iterations,
		Runtime:       float64(time.Since(start).Nanoseconds()),
		NumTargets:    len(simcfg.Targets),
		TargetDetails: simcfg.Targets,
		Config:        cfg,
	}
	result.Map(simcfg, stats)
	result.Text = result.PrettyPrint()

	charDetails, err := GenerateCharacterDetails(simcfg)
	if err != nil {
		return result, err
	}
	result.CharDetails = charDetails

	//run one debug
	//debug call will clone before running
	debugOut, err := GenerateDebugLogWithSeed(simcfg, CryptoRandSeed())
	if err != nil {
		return result, err
	}
	result.Debug = debugOut

	// Include debug logs for min/max-DPS runs if requested.
	if opts.DebugMinMax {
		minDPSDebugOut, err := GenerateDebugLogWithSeed(simcfg, int64(result.MinSeed))
		if err != nil {
			return result, err
		}
		result.DebugMinDPSRun = minDPSDebugOut

		maxDPSDebugOut, err := GenerateDebugLogWithSeed(simcfg, int64(result.MaxSeed))
		if err != nil {
			return result, err
		}
		result.DebugMaxDPSRun = maxDPSDebugOut
	}
	return result, nil
}

// cryptoRandSeed generates a random seed using crypo rand
func CryptoRandSeed() int64 {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		log.Panic("cannot seed math/rand package with cryptographically secure random number generator")
	}
	return int64(binary.LittleEndian.Uint64(b[:]))
}

var reImport = regexp.MustCompile(`(?m)^import "(.+)"$`)

// readConfig will load and read the config at specified path. Will resolve any import statements
// as well
func ReadConfig(fpath string) (string, error) {

	src, err := ioutil.ReadFile(fpath)
	if err != nil {
		return "", err
	}

	//check for imports
	var data strings.Builder

	rows := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	for _, row := range rows {
		match := reImport.FindStringSubmatch(row)
		if match != nil {
			//read import
			p := path.Join(path.Dir(fpath), match[1])
			src, err = ioutil.ReadFile(p)
			if err != nil {
				return "", err
			}

			data.WriteString(string(src))

		} else {
			data.WriteString(row)
			data.WriteString("\n")
		}
	}

	return data.String(), nil
}
