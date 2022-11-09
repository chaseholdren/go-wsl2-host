package hostsapi

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const hostspath = "C:/Windows/System32/drivers/etc/hosts"

// HostEntry data structure for IP and hostnames
type HostEntry struct {
	idx      int
	IP       string
	Hostname string
	Comment  string
}

// HostsAPI data structure
type HostsAPI struct {
	filter    string
	entries   map[string]*HostEntry
	remidxs   map[int]interface{}
}

func parseHostfileLine(idx int, line string) ([]*HostEntry, error) {
	if len(line) <= 0 {
		return nil, errors.New("invalid line")
	}
	line = strings.TrimSpace(line)
	if line[0] == '#' {
		return nil, errors.New("comment line")
	}
	fields := strings.Fields(line)
	var ip string
	var hostnames []string
	var comment string
	var commentidx int
	for i, f := range fields {
		if f == "" {
			continue
		}
		if f[0] == '#' { // inline comment
			commentidx = i + 1
			break // don't process any more
		}
		if i == 0 {
			ip = f
		} else {
			hostnames = append(hostnames, f)
		}
	}
	if commentidx > 0 {
		comment = strings.Join(fields[commentidx:], " ")
	}
	if ip == "" || len(hostnames) == 0 {
		return nil, fmt.Errorf("invalid fields for line: %q", line)
	}
	var entries []*HostEntry
	for _, hostname := range hostnames {
		entries = append(entries, &HostEntry{
			idx:      idx,
			IP:       ip,
			Hostname: hostname,
			Comment:  comment,
		})
	}

	return entries, nil
}

// func (h *HostsAPI) loadAndParse() error {
// 	scanner := bufio.NewScanner(h.hostsfile)
// 	idx := 0
// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		entries, err := parseHostfileLine(idx, line)
// 		idx++
// 		if err != nil {
// 			// log.Println(err) // debug
// 			continue
// 		}
// 		for _, e := range entries {
// 			if h.filter == "" || strings.Contains(e.Comment, h.filter) {
// 				h.entries[e.Hostname] = e
// 				h.remidxs[e.idx] = nil
// 			}
// 		}
// 	}
// 	h.hostsfile.Seek(0, 0)
// 	return nil
// }

// CreateAPI creates a new instance of the hosts file API
// Call Close() when finished
// `filter` proves ability to filter by string contains
func CreateAPI(filter string) (*HostsAPI, error) {
	h := &HostsAPI{
		filter:    filter,
		remidxs:   make(map[int]interface{}),
		entries:   make(map[string]*HostEntry),
	}
	return h, nil
}

// Entries returns parsed entries of host file
func (h *HostsAPI) Entries() map[string]*HostEntry {
	return h.entries
}

// RemoveEntry removes existing entry from hosts file
func (h *HostsAPI) RemoveEntry(hostname string) error {
	if _, exists := h.entries[hostname]; exists {
		delete(h.entries, hostname)
	} else {
		return fmt.Errorf("failed to remove, hostname does not exist: %s", hostname)
	}
	return nil
}

// AddEntry adds a new HostEntry
func (h *HostsAPI) AddEntry(entry *HostEntry) error {
	if _, exists := h.entries[entry.Hostname]; exists {
		return fmt.Errorf("failed to add entry, hostname already exists: %s", entry.Hostname)
	}

	h.entries[entry.Hostname] = entry

	return nil
}

// Write
func (h *HostsAPI) Write() error {

	var outbuf bytes.Buffer

	f, err := os.Open(hostspath)

	if err != nil {
		return fmt.Errorf("failed to open hosts file: %w", err)
	}

	// first remove all current entries
	// scanner := bufio.NewScanner(h.hostsfile)
	// for idx := 0; scanner.Scan() == true; idx++ {
	// 	line := scanner.Text()
	// 	if _, exists := h.remidxs[idx]; !exists {
	// 		outbuf.WriteString(line)
	// 		outbuf.WriteString("\r\n")
	// 	}
	// }

	// append entries to file
	// for _, e := range h.entries {
	// 	var comment string
	// 	if e.Comment != "" {
	// 		comment = fmt.Sprintf("    # %s", e.Comment)
	// 	}
	// 	outbuf.WriteString(fmt.Sprintf("%s %s%s\r\n", e.IP, e.Hostname))
	// }


	scanner := bufio.NewScanner(f)
	inSection := false
	for idx := 0; scanner.Scan() == true; idx++ {

		line := scanner.Text()
		line = strings.TrimSpace(line)

		if strings.Contains(line, "# go-wsl2-host's room - stay out") {
			inSection = true
			continue
		}

		if inSection && strings.Contains(line, "# ok now you can come in :)") {
			inSection = false
			continue
		}

		if !inSection {
			if len(line) == 0 {
				outbuf.WriteString(line)
				outbuf.WriteString("\r\n")
				continue
			}


			if line[0] == '#' {
				outbuf.WriteString(line)
				outbuf.WriteString("\r\n")
				continue
			}
		}

		entries, err := parseHostfileLine(idx, line)
		if err != nil {
			continue
		}

		if !inSection {
			// lineEntries := make([]HostEntry, 0)
			var lineEntries []*HostEntry
			mustRewrite := false
			for _, e := range entries {
				if !mustRewrite && h.entries[e.Hostname] != nil {
					mustRewrite = true
				} else {
					lineEntries = append(lineEntries, e)
				}
			}

			if mustRewrite && len(lineEntries) > 0 {
				hostNames := make([]string, len(lineEntries))
				for j, e := range lineEntries {
					hostNames[j] = e.Hostname
				}
				hostNameString := strings.Join(hostNames, " ")
				line = fmt.Sprintf("%s %s %s\r\n", lineEntries[0].IP, hostNameString, lineEntries[0].Comment)
			}

			outbuf.WriteString(line)
			outbuf.WriteString("\r\n")
			continue
		}

		for _, e := range entries {
			if h.entries[e.Hostname] == nil {
				h.entries[e.Hostname] = e
			}
		}

		idx++
	}

	outbuf.WriteString("# go-wsl2-host's room - stay out\r\n")

	for _, e := range h.entries {
		if len(e.IP) > 0 {
			var comment string
			if e.Comment != "" && e.Comment[0] != '#' {
				comment = fmt.Sprintf("    # %s", e.Comment)
			}
			comment = ""
			line := fmt.Sprintf("%s %s %s\r\n", e.IP, e.Hostname, comment)
			outbuf.WriteString(line)
		}
	}

	outbuf.WriteString("# ok now you can come in :)\r\n")

	// f.Seek(0, 0)

	f.Close()


	//

	f, err = os.OpenFile(hostspath, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open hosts file for writing: %w", err)
	}
	defer f.Close()

	f.Write(outbuf.Bytes())
	err = f.Sync()
	if err != nil {
		return err
	}

	return nil
}

// GetHostIP returns the IP address of Hyper-V Switch on the host connected to WSL
func GetHostIP() (string, error) {
	cmd := exec.Command("netsh", "interface", "ip", "show", "address", "vEthernet (WSL)") //, "|", "findstr", "IP Address", "|", "%", "{", "$_", "-replace", "IP Address:", "", "}", "|", "%", "{", "$_", "-replace", " ", "", "}")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// If system language not english, the output not "IP Address". such as in chinese it's "IP 地址".
	// And the output no have other such as "IP", so we can only match the "IP".
	ipRegex := regexp.MustCompile("IP.*:?\040*(.*)\r\n")
	ipString := ipRegex.FindStringSubmatch(string(out))
	if len(ipString) != 2 {
		return "", errors.New(`netsh interface ip show address "vEthernet (WSL)"`)
	}
	return ipString[1], nil
}
