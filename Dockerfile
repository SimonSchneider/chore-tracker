FROM gcr.io/distroless/static-debian11
COPY --chmod=555 bin/linux/app /
VOLUME /db
EXPOSE 80
ENTRYPOINT ["/app", "-addr", ":80", "-dburl", "file:/db/chores.sqlite?cache=shared"]
