ctx, span := otel.Tracer("app").Start(ctx, "myClass.MyFunction")
defer span.End()
defer func() {
	if err != nil {
		span.SetStatus(otelCodes.Error, "error")
		span.RecordError(err)
	}
}()