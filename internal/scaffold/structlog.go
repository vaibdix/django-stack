package scaffold

import (
	"context"
	"fmt"
)

func (e *Env) applyStructlog(ctx context.Context) error {
	se, err := e.settingsEditor(e.dir)
	if err != nil {
		return err
	}

	if err := se.AddToList(installedApps, appStructlog); err != nil {
		return fmt.Errorf("failed to add %s to %s: %w", appStructlog, installedApps, err)
	}

	if err := se.AddToList(middlewareList, middlewareStructlog); err != nil {
		return fmt.Errorf("failed to add %s to %s: %w", middlewareStructlog, middlewareList, err)
	}

	if err := se.AddImport("import structlog"); err != nil {
		return err
	}
	se.AppendBlock(e.structlogSettingsBlock())

	if err := se.Save(); err != nil {
		return err
	}

	e.Step("Structured logging configured (structlog + django-structlog).")
	e.Step("Every log line carries request_id; authenticated requests also carry user_id.")
	return nil
}

func (e *Env) structlogSettingsBlock() string {
	return fmt.Sprintf(`
_LOG_RENDERER = 'plain_console' if %s else 'json_formatter'

_FOREIGN_PRE_CHAIN = [
    structlog.contextvars.merge_contextvars,
    structlog.processors.TimeStamper(fmt='iso'),
    structlog.stdlib.add_logger_name,
    structlog.stdlib.add_log_level,
    structlog.stdlib.PositionalArgumentsFormatter(),
]

LOGGING = {
    'version': 1,
    'disable_existing_loggers': False,
    'formatters': {
        'json_formatter': {
            '()': structlog.stdlib.ProcessorFormatter,
            'processor': structlog.processors.JSONRenderer(),
            'foreign_pre_chain': _FOREIGN_PRE_CHAIN,
        },
        'plain_console': {
            '()': structlog.stdlib.ProcessorFormatter,
            'processor': structlog.dev.ConsoleRenderer(),
            'foreign_pre_chain': _FOREIGN_PRE_CHAIN,
        },
    },
    'handlers': {
        'console': {
            'class': 'logging.StreamHandler',
            'formatter': _LOG_RENDERER,
        },
    },
    'root': {
        'handlers': ['console'],
        'level': %s,
    },
    'loggers': {
        'django': {
            'handlers': ['console'],
            'level': 'INFO',
            'propagate': False,
        },
        'django_structlog': {
            'handlers': ['console'],
            'level': 'INFO',
            'propagate': False,
        },
        '%s': {
            'handlers': ['console'],
            'level': %s,
            'propagate': False,
        },
    },
}

structlog.configure(
    processors=[
        structlog.contextvars.merge_contextvars,
        structlog.stdlib.filter_by_level,
        structlog.processors.TimeStamper(fmt='iso'),
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.processors.UnicodeDecoder(),
        structlog.stdlib.ProcessorFormatter.wrap_for_formatter,
    ],
    logger_factory=structlog.stdlib.LoggerFactory(),
    cache_logger_on_first_use=True,
)
`,
		"DEBUG",
		logLevelExpression,
		e.cfg.ModuleName(),
		logLevelExpression,
	)
}

const logLevelExpression = "'DEBUG' if DEBUG else 'INFO'"
