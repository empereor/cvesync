package main

import (
	"github.com/blackjack/syslog"
	"github.com/mikkolehtisalo/cvesync/nvd"
	"github.com/mikkolehtisalo/cvesync/tracker"
	"github.com/mikkolehtisalo/cvesync/util"
)

var (
	config util.ServiceConfiguration
)

func sync(feed nvd.CVE, cwes nvd.CWE, ts tracker.Tracker) {
	db := util.Get_DB(config.DBFile)
	defer db.Close()

	// Initialize tracker
	ts.Init()

	for _, entry := range feed.Entries {
		entry.CWE.CWECatalog = &cwes
		// Completely new?
		if !util.Exists(db, entry.Id) {
			syslog.Noticef("Adding new CVE %s", entry.Id)
			id, err := ts.Add(entry)
			if err != nil {
				syslog.Errf("Unable to add %v to issue tracker: %v", entry.Id, err)
				continue
			}
			// Add to database, too
			util.DB_Add(db, entry.Id, entry.Last_Modified, id)
			// Already existing, but modified?
		} else if !util.Modified_Matches(db, entry.Id, entry.Last_Modified) {
			syslog.Noticef("Modifying old CVE %s", entry.Id)
			ticketid := util.DB_TicketID(db, entry.Id)
			err := ts.Update(entry, ticketid)
			if err != nil {
				syslog.Errf("Unable to modify %v in issue tracker: %v", entry.Id, err)
				continue
			}
			// Update to database, too
			util.DB_Update(db, entry.Id, entry.Last_Modified)
		}
	}
}

func main() {
	syslog.Openlog("cvesync", syslog.LOG_PID, syslog.LOG_DAEMON)
	syslog.Info("Cvesync started")

	config = util.Load_Config("/opt/cvesync/etc/settings.json")
	cve_feed := nvd.Get_CVE_feed(config.FeedURL, config.CAKeyFile)
	cwes := nvd.Get_CWEs(config.CWEfile)

	ts := tracker.Jira{}
	//ts := tracker.RT{}
	sync(cve_feed, cwes, &ts)

	syslog.Info("Cvesync ended")
}
