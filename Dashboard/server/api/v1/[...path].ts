type ProxyError = {
  message?: string
}

function normalizePath(path: unknown) {
  const rawSegments = Array.isArray(path)
    ? path.flatMap(segment => String(segment).split('/'))
    : typeof path === 'string' ? path.split('/') : []

  const segments: string[] = []

  for (const rawSegment of rawSegments) {
    if (!rawSegment) {
      continue
    }

    let segment: string
    try {
      segment = decodeURIComponent(rawSegment)
    } catch {
      return null
    }

    if (segment === '.' || segment === '..') {
      return null
    }

    segments.push(encodeURIComponent(segment))
  }

  return segments.join('/')
}

function isBackupDownloadPath(path: string) {
  return /^backups\/\d+\/download$/.test(path)
}

function hasAuthCredential(event: Parameters<typeof getMethod>[0]) {
  const cookie = getRequestHeader(event, 'cookie') || ''
  const authorization = getRequestHeader(event, 'authorization') || ''

  return cookie.includes('zenspanel_token=') || authorization.startsWith('Bearer ')
}

function guardBackupDownload(event: Parameters<typeof getMethod>[0], method: string) {
  if (method !== 'GET' && method !== 'HEAD') {
    setResponseHeader(event, 'allow', 'GET, HEAD')
    setResponseStatus(event, 405)
    return { error: 'method not allowed' }
  }

  if (!hasAuthCredential(event)) {
    setResponseStatus(event, 401)
    return { error: 'authentication required' }
  }

  return null
}

function getPeerIP(event: Parameters<typeof getMethod>[0]) {
  return event.node.req.socket.remoteAddress?.replace(/^::ffff:/, '') || ''
}

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const backendUrl = config.backendUrl

  // Extract the path after /api/v1/
  const path = normalizePath(event.context.params?.path)
  if (path === null) {
    setResponseStatus(event, 400)
    return { error: 'invalid path' }
  }

  const isBackupDownload = isBackupDownloadPath(path)

  const method = getMethod(event)
  const guardError = isBackupDownload ? guardBackupDownload(event, method) : null
  if (guardError) {
    return guardError
  }

  try {
    const body = method !== 'GET' && method !== 'HEAD'
      ? await readBody(event).catch(() => undefined)
      : undefined

    const query = getQuery(event)

    // Build headers to forward — pass relevant client headers to backend
    const forwardHeaders: Record<string, string> = {}

    // Forward content-type for POST/PUT/PATCH
    const contentType = getRequestHeader(event, 'content-type')
    if (contentType) {
      forwardHeaders['content-type'] = contentType
    }

    // Forward cookies from browser to backend (carries JWT from HttpOnly cookie)
    const cookie = getRequestHeader(event, 'cookie')
    if (cookie) {
      forwardHeaders.cookie = cookie
    }

    // Forward Authorization header if present (for API key auth, etc.)
    const authorization = getRequestHeader(event, 'authorization')
    if (authorization) {
      forwardHeaders.authorization = authorization
    }

    // Forward real client IP
    const clientIP = getPeerIP(event)
    if (clientIP) {
      forwardHeaders['x-forwarded-for'] = clientIP
      forwardHeaders['x-real-ip'] = clientIP
    }

    // Forward user-agent for audit logs
    const userAgent = getRequestHeader(event, 'user-agent')
    if (userAgent) {
      forwardHeaders['user-agent'] = userAgent
    }

    const target = `${backendUrl}/api/v1/${path}`

    // console.log(`[proxy] ${method} ${target}${query ? '?' + new URLSearchParams(query as Record<string, string>).toString() : ''}`)

    const response = await $fetch.raw(target, {
      method,
      headers: forwardHeaders,
      body,
      query,
      responseType: isBackupDownload ? 'arrayBuffer' : undefined,
      ignoreResponseError: true
    })

    // Forward Set-Cookie from backend to browser
    const setCookie = response.headers.get('set-cookie')
    if (setCookie) {
      setResponseHeader(event, 'set-cookie', setCookie)
    }

    if (isBackupDownload) {
      const contentDisposition = response.headers.get('content-disposition')
      const contentType = response.headers.get('content-type')
      const contentLength = response.headers.get('content-length')

      if (contentDisposition) {
        setResponseHeader(event, 'content-disposition', contentDisposition)
      }
      if (contentType) {
        setResponseHeader(event, 'content-type', contentType)
      }
      if (contentLength) {
        setResponseHeader(event, 'content-length', Number(contentLength))
      }

      setResponseHeader(event, 'cache-control', 'no-store')
      setResponseHeader(event, 'x-content-type-options', 'nosniff')
    }

    setResponseStatus(event, response.status)
    return response._data
  } catch (err: unknown) {
    // Network errors, connection refused, etc.
    const proxyError = err as ProxyError
    console.error(`[proxy] Error proxying to ${backendUrl}/api/v1/${path}:`, proxyError.message)
    setResponseStatus(event, 502)
    return { error: 'Backend service unavailable' }
  }
})
