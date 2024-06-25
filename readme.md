# Traefik IP2Location Plugin

A Traefik plugin that uses IP2Location to determine the locale based on the client's IP address and redirect
accordingly.

## Configuration in Traefik:

### Static configuration

```yaml
entryPoints:
  web:
    address: :80
    http:
      middlewares:
        - LocaleIp2Location@file

experimental:
  localPlugins:
    traefik-ip2location:
      moduleName: github.com/username/traefik-ip2location
```

### Dynamic configuration

The plugin requires the following configuration options:

- `dbPath`: Path to the IP2Location database file.
- `locales`: List of supported locales.
- `defaultLocale`: Default locale to use if the detected locale is not supported.
- `languageStrategy`: Strategy for adding the language to the request (header, path, or query). Default is 'header'.
- `languageParam`: Parameter name for the language in the URL (required if languageStrategy is query).
- `redirect`: (Optional) Boolean flag to enable redirection when the language is changed for path and query strategies. Default is false.
- `languageToCountriesOverride`: (Optional) A map to override the default language-to-countries mapping. This parameter
  has priority over the default mapping.

Using Header Strategy

```yaml
http:
  middlewares:
    LocaleIp2Location:
      plugin:
        traefik-ip2location:
          dbPath: "/path/to/IP2LOCATION-LITE-DB1.BIN"
          locales: [ "US", "CA", "GB" ]
          defaultLocale: "US"
          languageStrategy: "header"
          languageToCountriesOverride:
            en: [ "US", "GB" ]
            fr: [ "CA" ]
```

Using Path Strategy

```yaml
http:
  middlewares:
    LocaleIp2Location:
      plugin:
        traefik-ip2location:
          dbPath: "/path/to/IP2LOCATION-LITE-DB1.BIN"
          locales: [ "US", "CA", "GB" ]
          defaultLocale: "US"
          languageStrategy: "path"
          redirect: true
          languageToCountriesOverride:
            en: [ "US", "GB" ]
            fr: [ "CA" ]
```

Using Query Strategy

```yaml
http:
  middlewares:
    LocaleIp2Location:
      plugin:
        traefik-ip2location:
          dbPath: "/path/to/IP2LOCATION-LITE-DB1.BIN"
          locales: [ "US", "CA", "GB" ]
          defaultLocale: "US"
          languageStrategy: "query"
          languageParam: "lang"
          redirect: true
          languageToCountriesOverride:
            en: [ "US", "GB" ]
            fr: [ "CA" ]
```
