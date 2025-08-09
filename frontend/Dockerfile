FROM ghcr.io/cirruslabs/flutter AS build

WORKDIR /app
COPY . .
RUN flutter build web

FROM nginx:alpine
COPY --from=build /app/build/web /usr/share/nginx/html
# Symlink config folder for simpler binding
RUN rm -rf /usr/share/nginx/html/assets/assets/config \
    && mkdir -p /config \
    && ln -s /config /usr/share/nginx/html/assets/assets/config