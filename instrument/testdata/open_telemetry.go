ctx, span := otel.Tracer("app").Start(ctx, "myClass.MyFunction")
defer span.End()