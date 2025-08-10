FROM ghcr.io/cirruslabs/flutter AS build

WORKDIR /app
COPY . .
RUN flutter build web --pwa-strategy=none

FROM caddy:2
COPY --from=build /app/build/web /usr/share/caddy
# Symlink config folder for simpler binding
RUN rm -rf /usr/share/caddy/assets/assets/config \
    && mkdir -p /webconfig \
    && ln -s /webconfig /usr/share/caddy/assets/assets/config