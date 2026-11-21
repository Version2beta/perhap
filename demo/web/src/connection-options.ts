import { credsAuthenticator, type ConnectionOptions } from 'nats.ws'

// credentials is the contents of a NATS user .creds file, not a URL or token.
export function connectionOptions(server: string, credentials?: string): ConnectionOptions {
  return {
    servers: server,
    ...(credentials ? { authenticator: credsAuthenticator(new TextEncoder().encode(credentials)) } : {}),
  }
}
