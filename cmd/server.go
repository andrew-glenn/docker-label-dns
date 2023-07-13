// Copyright © 2017 uxbh
// This file is part of github.com/uxbh/ztdns.

package cmd

import (
	"os"
	"os/signal"

	"github.com/andrew-glenn/docker-compose-dns/dnssrv"
	"github.com/andrew-glenn/docker-compose-dns/dockerSocket"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run ztDNS server",
	Long: `Server (ztdns server) will start the DNS server.append
	
	Example: ztdns server`,
	Run: func(cmd *cobra.Command, args []string) {
		// Update the DNSDatabase
		req := make(chan string)
		// Start the DNS server
		go dnssrv.Start(viper.GetString("interface"), viper.GetInt("port"), viper.GetString("suffix"), req)

		dockerSocket.Start("unix:///var/run/docker.sock", dnssrv.DNSDatabase)

		// Start the Docker socket listener / introspector.

		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt)
	forever:
		for {
			select {
			case <-sig:
				log.Println("signal received, stopping")
				break forever
			default:
				n := <-req
				log.Infof("Got request for %s", n)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(serverCmd)
	serverCmd.PersistentFlags().String("interface", "", "interface to listen on")
	serverCmd.PersistentFlags().String("port", "", "interface to listen on")

	viper.BindPFlag("interface", serverCmd.PersistentFlags().Lookup("interface"))
	viper.BindPFlag("port", serverCmd.PersistentFlags().Lookup("port"))

}
