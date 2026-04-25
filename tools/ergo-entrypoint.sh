#!/bin/sh
# Based on https://github.com/ergochat/ergo/blob/master/distrib/docker/run.sh
# - Maps HOST_NAME to Ergo network.name and server.name (replaces ngircd --server-name)
set -e
HOST_NAME="${HOST_NAME:-cossacks-irc}"
export HOST_NAME

# When ERGO__NETWORK__NAME / ERGO__SERVER__NAME are set (docker-compose), Ergo applies them at runtime
# (allow-environment-overrides in default.yaml; see Ergo MANUAL.md). Else patch ircd.yaml from HOST_NAME.
patch_ircd_names() {
	awk 'BEGIN{h=ENVIRON["HOST_NAME"]}
		/^network:/{st=1; print; next}
		st==1 && /^[[:space:]]*name:/{ sub(/name:[[:space:]].*/, "name: " h); st=0; print; next }
		/^server:/{st=2; print; next}
		st==2 && /^[[:space:]]*name:/{ sub(/name:[[:space:]].*/, "name: " h); st=0; print; next }
		{print}' /ircd/ircd.yaml > /tmp/ircd.yaml
	mv /tmp/ircd.yaml /ircd/ircd.yaml
}

if [ ! -f /ircd/ircd.yaml ]; then
	awk '{gsub(/path: languages/,"path: /ircd-bin/languages")}1' /ircd-bin/default.yaml > /tmp/ircd.yaml
	OPERPASS=$(tr -dc _A-Z-a-z-0-9 < /dev/urandom | head -c 20)
	echo "Oper username:password is admin:$OPERPASS" >&2
	ENCRYPTEDPASS=$(printf '%s' "$OPERPASS" | /ircd-bin/ergo genpasswd)
	ORIGINALPASS='\$2a\$04\$0123456789abcdef0123456789abcdef0123456789abcdef01234'
	awk "{gsub(/password: \\\"$ORIGINALPASS\\\"/,\"password: \\\"$ENCRYPTEDPASS\\\"\")}1" /tmp/ircd.yaml > /tmp/ircd2.yaml
	unset OPERPASS ENCRYPTEDPASS ORIGINALPASS
	mv /tmp/ircd2.yaml /ircd/ircd.yaml
fi

if [ -z "$ERGO__SERVER__NAME" ] || [ -z "$ERGO__NETWORK__NAME" ]; then
	patch_ircd_names
fi

# `docker compose up` + Ctrl+C stops containers without removing them.
# On next `up`, rerunning mkcerts fails if certs already exist.
if [ -f /ircd/fullchain.pem ] && [ -f /ircd/privkey.pem ]; then
	:
elif [ ! -f /ircd/.ergo-certs-generated ]; then
	/ircd-bin/ergo mkcerts
	touch /ircd/.ergo-certs-generated
fi

exec /ircd-bin/ergo run
