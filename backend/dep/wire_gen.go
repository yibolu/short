// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package dep

import (
	"database/sql"
	"short/app/adapter/db"
	"short/app/adapter/facebook"
	"short/app/adapter/github"
	"short/app/adapter/graphql"
	"short/app/usecase/account"
	"short/app/usecase/requester"
	"short/app/usecase/url"
	"short/dep/provider"
	"time"

	"github.com/byliuyang/app/fw"
	"github.com/byliuyang/app/modern/mdcli"
	"github.com/byliuyang/app/modern/mddb"
	"github.com/byliuyang/app/modern/mdhttp"
	"github.com/byliuyang/app/modern/mdlogger"
	"github.com/byliuyang/app/modern/mdrequest"
	"github.com/byliuyang/app/modern/mdrouting"
	"github.com/byliuyang/app/modern/mdservice"
	"github.com/byliuyang/app/modern/mdtimer"
	"github.com/byliuyang/app/modern/mdtracer"
	"github.com/google/wire"
)

// Injectors from wire.go:

func InjectCommandFactory() fw.CommandFactory {
	cobraFactory := mdcli.NewCobraFactory()
	return cobraFactory
}

func InjectDBConnector() fw.DBConnector {
	postgresConnector := mddb.NewPostgresConnector()
	return postgresConnector
}

func InjectDBMigrationTool() fw.DBMigrationTool {
	postgresMigrationTool := mddb.NewPostgresMigrationTool()
	return postgresMigrationTool
}

func InjectGraphQlService(name string, sqlDB *sql.DB, graphqlPath provider.GraphQlPath, secret provider.ReCaptchaSecret, jwtSecret provider.JwtSecret, bufferSize provider.KeyGenBufferSize, kgsRPCConfig provider.KgsRPCConfig) (mdservice.Service, error) {
	logger := mdlogger.NewLocal()
	tracer := mdtracer.NewLocal()
	urlSql := db.NewURLSql(sqlDB)
	retrieverPersist := url.NewRetrieverPersist(urlSql)
	userURLRelationSQL := db.NewUserURLRelationSQL(sqlDB)
	rpc, err := provider.NewKgsRPC(kgsRPCConfig)
	if err != nil {
		return mdservice.Service{}, err
	}
	remote, err := provider.NewRemote(bufferSize, rpc)
	if err != nil {
		return mdservice.Service{}, err
	}
	creatorPersist := url.NewCreatorPersist(urlSql, userURLRelationSQL, remote)
	client := mdhttp.NewClient()
	http := mdrequest.NewHTTP(client)
	reCaptcha := provider.NewReCaptchaService(http, secret)
	verifier := requester.NewVerifier(reCaptcha)
	cryptoTokenizer := provider.NewJwtGo(jwtSecret)
	timer := mdtimer.NewTimer()
	tokenValidDuration := _wireTokenValidDurationValue
	authenticator := provider.NewAuthenticator(cryptoTokenizer, timer, tokenValidDuration)
	short := graphql.NewShort(logger, tracer, retrieverPersist, creatorPersist, verifier, authenticator)
	server := provider.NewGraphGophers(graphqlPath, logger, tracer, short)
	service := mdservice.New(name, server, logger)
	return service, nil
}

var (
	_wireTokenValidDurationValue = provider.TokenValidDuration(oneDay)
)

func InjectRoutingService(name string, sqlDB *sql.DB, githubClientID provider.GithubClientID, githubClientSecret provider.GithubClientSecret, facebookClientID provider.FacebookClientID, facebookClientSecret provider.FacebookClientSecret, facebookRedirectURI provider.FacebookRedirectURI, jwtSecret provider.JwtSecret, webFrontendURL provider.WebFrontendURL) mdservice.Service {
	logger := mdlogger.NewLocal()
	tracer := mdtracer.NewLocal()
	timer := mdtimer.NewTimer()
	urlSql := db.NewURLSql(sqlDB)
	retrieverPersist := url.NewRetrieverPersist(urlSql)
	client := mdhttp.NewClient()
	http := mdrequest.NewHTTP(client)
	identityProvider := provider.NewGithubIdentityProvider(http, githubClientID, githubClientSecret)
	graphQlRequest := mdrequest.NewGraphQl(http)
	githubAccount := github.NewAccount(graphQlRequest)
	api := github.NewAPI(identityProvider, githubAccount)
	facebookIdentityProvider := provider.NewFacebookIdentityProvider(http, facebookClientID, facebookClientSecret, facebookRedirectURI)
	facebookAccount := facebook.NewAccount()
	facebookAPI := facebook.NewAPI(facebookIdentityProvider, facebookAccount)
	cryptoTokenizer := provider.NewJwtGo(jwtSecret)
	tokenValidDuration := _wireTokenValidDurationValue
	authenticator := provider.NewAuthenticator(cryptoTokenizer, timer, tokenValidDuration)
	userSQL := db.NewUserSQL(sqlDB)
	repoService := account.NewRepoService(userSQL, timer)
	v := provider.NewShortRoutes(logger, tracer, webFrontendURL, timer, retrieverPersist, api, facebookAPI, authenticator, repoService)
	server := mdrouting.NewBuiltIn(logger, tracer, v)
	service := mdservice.New(name, server, logger)
	return service
}

// wire.go:

const oneDay = 24 * time.Hour

var authSet = wire.NewSet(provider.NewJwtGo, wire.Value(provider.TokenValidDuration(oneDay)), provider.NewAuthenticator)

var observabilitySet = wire.NewSet(mdlogger.NewLocal, mdtracer.NewLocal)

var githubAPISet = wire.NewSet(provider.NewGithubIdentityProvider, github.NewAccount, github.NewAPI)

var facebookAPISet = wire.NewSet(provider.NewFacebookIdentityProvider, facebook.NewAccount, facebook.NewAPI)
