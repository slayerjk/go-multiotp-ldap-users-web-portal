package ldapwork

import (
	"crypto/tls"
	"fmt"

	"github.com/go-ldap/ldap/v3"
)

// Make LDAP connection(without TLS), using Domain's FQDN & LDAP's common port(389)
func MakeLdapConnection(ldapFqdn string) (*ldap.Conn, error) {
	// defining connection URL: ldap/ldaps
	connURL := fmt.Sprintf("ldap://%s:389", ldapFqdn)

	// dial URL
	conn, err := ldap.DialURL(connURL)
	if err != nil {
		return nil, err
	}

	// if debug level neede
	// conn.Debug = true

	return conn, nil
}

// Make LDAP TLS Connection with existing LDAP connection
// Start connect with default ldap conn(389) then reconnect to use TLS
func StartTLSConnWoVerification(conn *ldap.Conn) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	// reDial with TLS
	err := conn.StartTLS(tlsConfig)
	if err != nil {
		return err
	}

	return nil
}

// Make LDAP Bind
func LdapBind(conn *ldap.Conn, bindUser, bindPassword string) error {
	err := conn.Bind(bindUser, bindPassword)
	if err != nil {
		return err
	}

	return nil
}

// Make LDAP search request based on LDAP filter & LDAP Attributes to get
// Example of filter: "(&(objectClass=user)(samaccountname=%s))"
func MakeSearchReq(conn *ldap.Conn, ldapBaseDN string, ldapFilter string, ldapAttrs ...string) ([]*ldap.Entry, error) {
	searchReq := ldap.NewSearchRequest(
		ldapBaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		ldapFilter,
		ldapAttrs,
		nil,
	)

	// making LDAP search request
	conResult, err := conn.Search(searchReq)
	if err != nil {
		return nil, err
	}

	// check if result is empty
	if len(conResult.Entries) == 0 {
		return nil, fmt.Errorf("empty result")
	}

	return conResult.Entries, nil
}

// Search ONLY Enabled user's 'samaccountname' by it's 'displayname'
//
// filterName - PREFIX of samaAccountName to exclude from result, use "" if don't need it;
// also exclude disabled accounts (userAccountControl != 546|514);
// dnFilter - some text of full DN to INCLUDE even if account is disabled, use "" if don't need it;
func SearchEnabledSAMAByDisplayname(conn *ldap.Conn, displayName, ldapBasedn, filterSama string) (string, error) {
	var result string

	// forming LDAP filter; use exclude prefix if len(filterSama) > 0
	filter := fmt.Sprintf("(&(objectClass=user)(displayname=%s))", displayName)
	if len(filterSama) > 0 {
		filter = fmt.Sprintf("(&(objectClass=user)(displayname=%s)(!samaccountname=%s*))", displayName, filterSama)
	}

	// forming LDAP search request for 'samaccountname' and 'useraccountcontrol'(disabled=546)
	searchResult, err := MakeSearchReq(conn, ldapBasedn, filter, "samaccountname", "useraccountcontrol")
	if err != nil {
		return "", fmt.Errorf("failed to make LDAP search:\n\t%v", err)
	}

	// check if len(conResult.Entries) > 1, choose only enabled(userAccountControl != 546)
	if len(searchResult) >= 1 {
		for ind := range searchResult {
			// pretty print
			// conResult.Entries[ind].PrettyPrint(4)

			// choose first enabled account as result
			userAccountControl := searchResult[ind].GetAttributeValue("userAccountControl")

			if userAccountControl != "546" && userAccountControl != "514" {
				result = searchResult[ind].GetAttributeValue("sAMAccountName")

				if len(result) == 0 {
					return "", fmt.Errorf("failed to find 'sAMAccountName' attribute value, empty result")
				}
				return result, nil
			}
		}
	}

	return "", fmt.Errorf("failed to find any ENABLED account entry")
}
