export class ApiError extends Error {
  status: number
  code: string

  constructor(message: string, status: number, code: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

export class ApiNotFoundError extends ApiError {
  constructor(message = 'Resource not found') {
    super(message, 404, 'not_found')
    this.name = 'ApiNotFoundError'
  }
}

export class ApiValidationError extends ApiError {
  fields: Record<string, string>

  constructor(message: string, fields: Record<string, string> = {}) {
    super(message, 400, 'validation_error')
    this.name = 'ApiValidationError'
    this.fields = fields
  }
}

export class ApiNetworkError extends ApiError {
  constructor(message = 'Network error') {
    super(message, 0, 'network_error')
    this.name = 'ApiNetworkError'
  }
}
