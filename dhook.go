/*
Package dhook provides a client for sending messages to Discord webhooks.

The client was specifically designed to allow sending a high volume of messages
without being rate limited by the Discord API (i.e. 429 response).
This is achieved by respecting the all rate limits when a request is sent to Discord:
  - Global rate limit: The global rate limit as specified in the official API documentation
  - Per-route rate limit: A dynamic rate limit taken given in the response header
  - Webhook rate limit: An undocumented rate limit specific to webhooks

Should the client still get rate limited it will block further requests to Discord
for the time the rate limit is in effect to prevent further escalation.
*/
package dhook
