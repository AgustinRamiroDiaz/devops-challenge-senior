I've left some TODOs about other things that could be added/improved

# Add e2e tests

- curl it and get current ip
- crash it and check the probes

# Add docker compose

# Improve ip detection

Important security caveat

Do not blindly trust the first value for authentication, authorization, rate-limit bypasses, or allowlists. A client can send its own X-Forwarded-For value, and some Google load-balancing configurations preserve the supplied portion before appending trusted values:

X-Forwarded-For: [supplied values], client-ip, load-balancer-ip

Therefore, the trustworthy entries are generally parsed from right to left, based on how many trusted proxies are in front of Cloud Run.

For a Cloud Run service exposed directly through its run.app URL, a practical approach is often:

func cloudRunClientIP(r \*http.Request) string {
parts := strings.Split(r.Header.Get("X-Forwarded-For"), ",")

    // Google commonly appends:
    //   ..., client-ip, proxy/load-balancer-ip
    if len(parts) >= 2 {
    	return strings.TrimSpace(parts[len(parts)-2])
    }

    return ""

}

However, the exact position depends on whether you also use Cloudflare, Firebase Hosting, API Gateway, an external Application Load Balancer, or another proxy. The robust solution is to define the trusted proxy chain and walk backward until reaching the first untrusted address.

Also note that this identifies the client’s internet-facing source IP, which could be a NAT gateway, corporate proxy, VPN, mobile carrier gateway, or CDN—not necessarily the individual device.
