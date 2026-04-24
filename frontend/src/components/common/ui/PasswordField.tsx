import { useState } from 'react'
import { IconButton, InputAdornment, TextField, Tooltip, type TextFieldProps } from '@mui/material'
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined'
import VisibilityOffOutlinedIcon from '@mui/icons-material/VisibilityOffOutlined'

type PasswordFieldProps = Omit<TextFieldProps, 'type'>

/**
 * TextField preset for password inputs. Renders a trailing eye icon that
 * toggles between masked and plain text. The visibility state is local —
 * consumers never need to manage it. Forwards every other TextField prop
 * so `required`, `autoComplete`, `helperText`, etc. keep working.
 */
export function PasswordField(props: PasswordFieldProps) {
  const [visible, setVisible] = useState(false)
  const { InputProps, inputProps, ...rest } = props

  return (
    <TextField
      {...rest}
      type={visible ? 'text' : 'password'}
      inputProps={{
        ...inputProps,
        // Ensure the browser treats both states as the same field for autofill.
        'data-lpignore': inputProps?.['data-lpignore'] ?? 'false',
      }}
      InputProps={{
        ...InputProps,
        endAdornment: (
          <InputAdornment position="end">
            <Tooltip title={visible ? 'Hide password' : 'Show password'} placement="left">
              <IconButton
                aria-label={visible ? 'Hide password' : 'Show password'}
                onClick={() => setVisible((v) => !v)}
                onMouseDown={(e) => e.preventDefault()}
                edge="end"
                size="small"
                sx={{ color: 'text.secondary', mr: 0.25 }}
              >
                {visible ? (
                  <VisibilityOffOutlinedIcon fontSize="small" />
                ) : (
                  <VisibilityOutlinedIcon fontSize="small" />
                )}
              </IconButton>
            </Tooltip>
          </InputAdornment>
        ),
      }}
    />
  )
}
