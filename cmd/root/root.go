package root

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Nerufa/go-shared/config"
	"github.com/Nerufa/go-shared/entrypoint"
	"github.com/Nerufa/go-shared/invoker"
	"github.com/Nerufa/go-shared/logger"
	"github.com/ProtocolONE/s3-proxy/cmd"
	"github.com/ProtocolONE/s3-proxy/cmd/version"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/automaxprocs/maxprocs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	configFile    string
	debug         bool
	gracefulDelay time.Duration
	initial       = config.Initial{}
	log           logger.Logger
	ep            entrypoint.Master
	e             error
	c             func()
)

const (
	prefix               = "cmd.root"
	envPrefix            = "APP"
	envWorkDir           = "APP_WD"
	viperCfgType         = "yaml"
	defaultGracefulDelay = 50 * time.Millisecond
)

// http://www.patorjk.com/software/taag/#p=display&f=Big&t=S3%20Proxy%20Server

var logo = `
   _____ ____    _____                        _____                          
  / ____|___ \  |  __ \                      / ____|                         
 | (___   __) | | |__) | __ _____  ___   _  | (___   ___ _ ____   _____ _ __ 
  \___ \ |__ <  |  ___/ '__/ _ \ \/ / | | |  \___ \ / _ \ '__\ \ / / _ \ '__|
  ____) |___) | | |   | | | (_) >  <| |_| |  ____) |  __/ |   \ V /  __/ |   
 |_____/|____/  |_|   |_|  \___/_/\_\\__, | |_____/ \___|_|    \_/ \___|_|   
                                      __/ |                                  
                                     |___/                                   

		VERSION: %v`

// Root command
var rootCmd = &cobra.Command{
	Use:           "bin [command]",
	Long:          "",
	Short:         fmt.Sprintf(logo, version.Version()),
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(subCmd *cobra.Command, _ []string) error {

		// initializing
		initial.WorkDir = os.Getenv(envWorkDir)
		if len(initial.WorkDir) == 0 {
			initial.WorkDir, e = filepath.Abs(filepath.Dir(os.Args[0]))
			if e != nil {
				return e
			}
		}
		initial.WorkDir, e = filepath.Abs(initial.WorkDir)
		if e != nil {
			return e
		}

		// bin pflags to viper
		e = initial.Viper.BindPFlags(subCmd.Parent().PersistentFlags())
		if e != nil {
			return e
		}
		e = initial.Viper.BindPFlags(subCmd.PersistentFlags())
		if e != nil {
			return e
		}

		initial.Viper.SetConfigFile(configFile)

		if configFile != "" {
			e := initial.Viper.ReadInConfig()
			if e != nil {
				return fmt.Errorf("can't read config, %v", errors.WithMessage(e, prefix))
			}
		}

		inv := invoker.NewInvoker()
		cmd.Observer = inv

		ep, c, e = entrypoint.Build(context.Background(), initial, inv)
		if e != nil {
			return e
		}
		defer c()

		cmd.Slave = ep
		log = ep.Logger().WithFields(logger.Fields{"service": prefix})

		go func() {
			reloadSignal := make(chan os.Signal, 1)
			signal.Notify(reloadSignal, syscall.SIGHUP)
			for {
				sig := <-reloadSignal
				inv.Reload(context.Background())
				ep.Logger().Info("OS signaled `%v`, reload", logger.Args(sig.String()))
			}
		}()

		go func() {
			shutdownSignal := make(chan os.Signal, 1)
			signal.Notify(shutdownSignal, syscall.SIGTERM, syscall.SIGINT)
			sig := <-shutdownSignal
			ep.Logger().Info("OS signaled `%v`, graceful shutdown in %s", logger.Args(sig.String(), gracefulDelay))
			ctx, _ := context.WithTimeout(context.Background(), gracefulDelay)
			ep.Shutdown(ctx, 0)
		}()

		return nil
	},
	PersistentPostRun: func(_ *cobra.Command, _ []string) {

		preRun := func() error {
			if debug {
				fmt.Printf(logo, version.Version())
				fmt.Println(color.RedString("\n\n# DEBUG INFO\n"))
				fmt.Printf("\nWork directory: %v\n\n", ep.WorkDir())
				fmt.Println(color.GreenString("# CONFIG FILE SETTINGS\n\n"))
				b, _ := json.Marshal(initial.Viper.AllSettings())
				var out bytes.Buffer
				e := json.Indent(&out, b, "", "  ")
				if e != nil {
					log.Error("can't prettify config")
					os.Exit(1)
				}
				fmt.Println(out.String())
				fmt.Println(color.CyanString("\n# LOGS\n\n"))
			}
			_, err := maxprocs.Set(maxprocs.Logger(log.Printf))
			return err
		}

		if e = ep.Serve(preRun); e != nil {
			_ = preRun()
			log.Error(e.Error())
			ep.Shutdown(context.Background(), 1)
		}
	},
}

func init() {
	initial.Viper = viper.New()
	initial.Viper.SetConfigType(viperCfgType)
	initial.Viper.SetEnvPrefix(envPrefix)
	initial.Viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	initial.Viper.AutomaticEnv()
	// pflags
	rootCmd.PersistentFlags().StringVarP(&configFile, config.UnmarshalKeyConfigFile, "c", "", "config file")
	rootCmd.PersistentFlags().BoolVarP(&debug, config.UnmarshalKeyDebug, "d", false, "debug mode")
	rootCmd.PersistentFlags().StringP(logger.UnmarshalKeyLevel, "l", "info", "logger level")
	rootCmd.PersistentFlags().StringSliceP(logger.UnmarshalKeyDebugTags, "t", []string{}, "logger tags for filter output, e.g.: -t tag -t tag2 -t key:value")
	rootCmd.PersistentFlags().DurationVar(&gracefulDelay, config.UnmarshalKeyGracefulDelay, defaultGracefulDelay, "graceful delay")
}

func Execute(cmds ...*cobra.Command) {
	rootCmd.AddCommand(cmds...)
	if e := rootCmd.Execute(); e != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", e.Error())
		os.Exit(1)
	}
}
