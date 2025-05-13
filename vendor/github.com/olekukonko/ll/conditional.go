package ll

// Conditional enables conditional logging based on a boolean condition.
// It wraps a logger with a condition that determines whether logging operations are executed,
// optimizing performance by skipping expensive operations (e.g., field computation, message formatting)
// when the condition is false. The struct supports fluent chaining for adding fields and logging.
type Conditional struct {
	logger    *Logger // Associated logger instance for logging operations
	condition bool    // Whether logging is allowed (true to log, false to skip)
}

// If creates a conditional logger that logs only if the condition is true.
// It returns a Conditional struct that wraps the logger, enabling conditional logging methods.
// This method is typically called on a Logger instance to start a conditional chain.
// Thread-safe via the underlying logger’s mutex.
// Example:
//
//	logger := New("app").Enable()
//	logger.If(true).Info("Logged")   // Output: [app] INFO: Logged
//	logger.If(false).Info("Ignored") // No output
func (l *Logger) If(condition bool) *Conditional {
	return &Conditional{logger: l, condition: condition}
}

// IfOne creates a conditional logger that logs only if all conditions are true.
// It evaluates a variadic list of boolean conditions, setting the condition to true only if
// all are true (logical AND). Returns a new Conditional with the result. Thread-safe via the
// underlying logger.
// Example:
//
//	logger := New("app").Enable()
//	logger.IfOne(true, true).Info("Logged")  // Output: [app] INFO: Logged
//	logger.IfOne(true, false).Info("Ignored") // No output
func (cl *Conditional) IfOne(conditions ...bool) *Conditional {
	result := true
	// Check each condition; set result to false if any is false
	for _, cond := range conditions {
		if !cond {
			result = false
			break
		}
	}
	return &Conditional{logger: cl.logger, condition: result}
}

// IfAny creates a conditional logger that logs only if at least one condition is true.
// It evaluates a variadic list of boolean conditions, setting the condition to true if any
// is true (logical OR). Returns a new Conditional with the result. Thread-safe via the
// underlying logger.
// Example:
//
//	logger := New("app").Enable()
//	logger.IfAny(false, true).Info("Logged")  // Output: [app] INFO: Logged
//	logger.IfAny(false, false).Info("Ignored") // No output
func (cl *Conditional) IfAny(conditions ...bool) *Conditional {
	result := false
	// Check each condition; set result to true if any is true
	for _, cond := range conditions {
		if cond {
			result = true
			break
		}
	}
	return &Conditional{logger: cl.logger, condition: result}
}

// Fields starts a fluent chain for adding fields using variadic key-value pairs, if the condition is true.
// It returns a FieldBuilder to attach fields, skipping field processing if the condition is false
// to optimize performance. Thread-safe via the FieldBuilder’s logger.
// Example:
//
//	logger := New("app").Enable()
//	logger.If(true).Fields("user", "alice").Info("Logged") // Output: [app] INFO: Logged [user=alice]
//	logger.If(false).Fields("user", "alice").Info("Ignored") // No output, no field processing
func (cl *Conditional) Fields(pairs ...any) *FieldBuilder {
	// Skip field processing if condition is false
	if !cl.condition {
		return &FieldBuilder{logger: cl.logger, fields: nil}
	}
	// Delegate to logger’s Fields method
	return cl.logger.Fields(pairs...)
}

// Field starts a fluent chain for adding fields from a map, if the condition is true.
// It returns a FieldBuilder to attach fields from a map, skipping processing if the condition
// is false. Thread-safe via the FieldBuilder’s logger.
// Example:
//
//	logger := New("app").Enable()
//	logger.If(true).Field(map[string]interface{}{"user": "alice"}).Info("Logged") // Output: [app] INFO: Logged [user=alice]
//	logger.If(false).Field(map[string]interface{}{"user": "alice"}).Info("Ignored") // No output
func (cl *Conditional) Field(fields map[string]interface{}) *FieldBuilder {
	// Skip field processing if condition is false
	if !cl.condition {
		return &FieldBuilder{logger: cl.logger, fields: nil}
	}
	// Delegate to logger’s Field method
	return cl.logger.Field(fields)
}

// Info logs a message at Info level if the condition is true.
// It formats the message using the provided format string and arguments, delegating to the
// logger’s Info method if the condition is true. Skips processing if false, optimizing performance.
// Thread-safe via the logger’s log method.
// Example:
//
//	logger := New("app").Enable()
//	logger.If(true).Info("Logged")   // Output: [app] INFO: Logged
//	logger.If(false).Info("Ignored") // No output
func (cl *Conditional) Info(format string, args ...any) {
	// Skip logging if condition is false
	if !cl.condition {
		return
	}
	// Delegate to logger’s Info method
	cl.logger.Info(format, args...)
}

// Debug logs a message at Debug level if the condition is true.
// It formats the message and delegates to the logger’s Debug method if the condition is true.
// Skips processing if false. Thread-safe via the logger’s log method.
// Example:
//
//	logger := New("app").Enable().Level(lx.LevelDebug)
//	logger.If(true).Debug("Logged")   // Output: [app] DEBUG: Logged
//	logger.If(false).Debug("Ignored") // No output
func (cl *Conditional) Debug(format string, args ...any) {
	// Skip logging if condition is false
	if !cl.condition {
		return
	}
	// Delegate to logger’s Debug method
	cl.logger.Debug(format, args...)
}

// Warn logs a message at Warn level if the condition is true.
// It formats the message and delegates to the logger’s Warn method if the condition is true.
// Skips processing if false. Thread-safe via the logger’s log method.
// Example:
//
//	logger := New("app").Enable()
//	logger.If(true).Warn("Logged")   // Output: [app] WARN: Logged
//	logger.If(false).Warn("Ignored") // No output
func (cl *Conditional) Warn(format string, args ...any) {
	// Skip logging if condition is false
	if !cl.condition {
		return
	}
	// Delegate to logger’s Warn method
	cl.logger.Warn(format, args...)
}

// Error logs a message at Error level if the condition is true.
// It formats the message and delegates to the logger’s Error method if the condition is true.
// Skips processing if false. Thread-safe via the logger’s log method.
// Example:
//
//	logger := New("app").Enable()
//	logger.If(true).Error("Logged")   // Output: [app] ERROR: Logged
//	logger.If(false).Error("Ignored") // No output
func (cl *Conditional) Error(format string, args ...any) {
	// Skip logging if condition is false
	if !cl.condition {
		return
	}
	// Delegate to logger’s Error method
	cl.logger.Error(format, args...)
}

// Stack logs a message at Error level with a stack trace if the condition is true.
// It formats the message and delegates to the logger’s Stack method with a stack trace if
// the condition is true. Skips processing if false. Thread-safe via the logger’s log method.
// Example:
//
//	logger := New("app").Enable()
//	logger.If(true).Stack("Logged")   // Output: [app] ERROR: Logged [stack=...]
//	logger.If(false).Stack("Ignored") // No output
func (cl *Conditional) Stack(format string, args ...any) {
	// Skip logging if condition is false
	if !cl.condition {
		return
	}
	// Delegate to logger’s Stack method
	cl.logger.Stack(format, args...)
}
