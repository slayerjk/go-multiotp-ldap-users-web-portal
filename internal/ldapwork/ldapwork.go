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
