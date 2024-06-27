# Traefik IP2Location Plugin

The Traefik IP2Location Plugin is a middleware for Traefik that allows automatic language detection based on the
client's IP address using the IP2Location database. This plugin can be configured to set the language for incoming
requests based on the detected location, and it supports various strategies for language detection and handling.

## Configuration

### Plugin Configuration

The plugin configuration is defined in the `Config` struct, which includes the following fields:

- **DBPath**: The path to the IP2Location database file. This file is used to determine the location based on the
  client's IP address.
- **Languages**: A list of supported languages. The plugin will use this list to validate and set the language for
  incoming requests.
- **DefaultLanguage**: The default language to use if the detected language is not supported or if the client's location
  cannot be determined.
- **LanguageParam** (optional, default: `lang`): The parameter name to use when the `query` strategy is selected. This
  parameter will be used to set the language to the query string.
- **LanguageStrategy** (optional, default: `header`): The strategy to use for handling the language from the request.
  Possible values are `header`, `path`, and `query`.
- **RedirectAfterHandling** (optional, default: `false`): A boolean flag that
  determines whether to perform a redirect after handling the language. If set to `true`, the plugin will redirect the
  client to the same URL with the updated language, actual for `path` and `query` strategies.
- **LanguageToCountriesOverride** (optional, default: `null`): A map that allows overriding the default language detection based on
  specific countries. The keys are language codes, and the values are lists of country codes.
- **DefaultLanguageHandling** (optional, default: `false`): A boolean flag that determines whether to handle requests
  with the default language. If set to `true`, requests with the default language will be processed; otherwise, they
  will be ignored.

#### **Language Strategies**

The plugin supports three strategies for handling the language from the request:

- **header**: The language is handling from the Accept-Language header.
- **path**: The language is handling from the URL path.
- **query**: The language is handling from the query string parameter specified by languageParam.

#### **Redirect After Handling**

If RedirectAfterHandling is set to true, the plugin will perform a redirect to the same URL with the updated language
after handling the request. This ensures that the client sees the updated language in the URL (actual for `path`
and `query` strategies).

#### **Overriding Language Detection**

The LanguageToCountriesOverride map allows you to override the default language detection for specific countries. This
can be useful if you want to enforce a particular language for users from certain countries.

#### **Default Language Handling**

The `DefaultLanguageHandling` parameter is a boolean flag that determines whether to handle requests with the default
language. When set to `true`, the plugin will process requests even if the detected language is the default language
specified in the configuration. This can be particularly useful when the default language of your website does not
require language-specific URLs, as it allows you to avoid modifying the URL for the default language.

For example, if your website's default language is English and your URLs are structured without a language prefix (
e.g., `example.com/about`), setting `DefaultLanguageHandling` to `true` ensures that requests from users whose detected
language is English will not be redirected to a language-specific URL (e.g., `example.com/en/about`). This maintains the
clean URL structure for the default language, providing a consistent user experience for visitors who use the default
language.

Additionally, the plugin will not make any changes to the request if the user's request already contains a language.
This ensures that the user's preference is respected and that the URL structure
remains consistent with the user's choice.

### Example Configuration

```yaml
http:
  middlewares:
    LocaleIp2Location:
      plugin:
        traefik-ip2location:
          dbPath: "/plugins-local/src/github.com/bublicov/traefik-ip2location/IP2LOCATION-LITE-DB1.BIN"
          languages: [ "en", "fr", "de" ]
          defaultLanguage: "en"
```

## Installation

To use the Traefik IP2Location Plugin, you need to install it as a **LOCAL PLUGIN** for Traefik. Here are the steps to
do
this:

1. **Clone the Plugin Repository**: Clone the repository of the Traefik IP2Location Plugin to your local path
   {root_traefik_dir}/plugins-local/src/github.com/bublicov/traefik-ip2location

    ```sh
    git clone https://github.com/bublicov/traefik-ip2location.git
    ```

2. **Static configuration**: Modify your Traefik configuration to include the local plugin. Here is an example of how to
   do
   this in your `traefik.yml` file:

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
          moduleName: github.com/bublicov/traefik-ip2location
    ```

3. **Dynamic Configuration**: Create a `dynamic.yml` file to define the middleware configuration for the plugin.

    ```yaml   
    #Header Strategy   
    http:
      middlewares:
        LocaleIp2Location:
          plugin:
            traefik-ip2location:
              dbPath: "/plugins-local/src/github.com/bublicov/traefik-ip2location/IP2LOCATION-LITE-DB1.BIN"
              languages: ["en", "fr", "de"]
              defaultLanguage: "en"
              languageToCountriesOverride: #optional
                fr: ["CA"]
    ```

    ```yaml   
    #Path Strategy   
    http:
      middlewares:
        LocaleIp2Location:
          plugin:
            traefik-ip2location:
              dbPath: "/plugins-local/src/github.com/bublicov/traefik-ip2location/IP2LOCATION-LITE-DB1.BIN"
              languages: ["en", "fr", "de"]
              defaultLanguage: "en"
              defaultLanguageHandling: false #optional (default: false)
              languageStrategy: "path"
              redirectAfterHandling: true #optional (default: false)
              languageToCountriesOverride: #optional
                fr: ["CA"]
    ```

    ```yaml   
    #Query Strategy   
    http:
      middlewares:
        LocaleIp2Location:
          plugin:
            traefik-ip2location:
              dbPath: "/plugins-local/src/github.com/bublicov/traefik-ip2location/IP2LOCATION-LITE-DB1.BIN"
              languages: ["en", "fr", "de"]
              defaultLanguage: "en"
              defaultLanguageHandling: false #optional (default: false)
              languageStrategy: "query"
              languageParam: "lg" #optional (default: lang)
              redirectAfterHandling: true #optional (default: false)
              languageToCountriesOverride: #optional
                fr: ["CA"]
    ```

### License

This plugin is licensed under the MIT License. See the LICENSE file for more details.
