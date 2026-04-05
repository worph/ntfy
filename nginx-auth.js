// njs module for ntfy login/auth cookie handling

function baseUrl(r) {
    var scheme = r.headersIn['X-Forwarded-Proto'] || r.variables.scheme || 'http';
    var host = r.headersIn['Host'] || 'localhost';
    return scheme + '://' + host;
}

function getAuthHeader(r) {
    // If client already sent Authorization header (mobile app, API client), use it
    var auth = r.headersIn['Authorization'];
    if (auth) {
        return auth;
    }
    // Otherwise try to extract from cookie
    var cookie = r.variables.cookie_ntfy_auth;
    if (cookie) {
        return 'Basic ' + cookie;
    }
    return '';
}

function checkCookie(r) {
    var cookie = r.variables.cookie_ntfy_auth;
    var auth = r.headersIn['Authorization'];
    if (cookie || auth) {
        r.return(200);
    } else {
        r.return(401);
    }
}

function loginPost(r) {
    var body = r.requestText || '';
    var params = {};
    body.split('&').forEach(function(pair) {
        var kv = pair.split('=');
        if (kv.length === 2) {
            params[decodeURIComponent(kv[0])] = decodeURIComponent(kv[1].replace(/\+/g, ' '));
        }
    });

    var username = params['username'] || '';
    var password = params['password'] || '';

    if (!username || !password) {
        r.headersOut['Location'] = baseUrl(r) + '/login?error=1';
        r.status = 302;
        r.sendHeader();
        r.finish();
        return;
    }

    // Base64 encode credentials
    var creds = username + ':' + password;
    var encoded = Buffer.from(creds).toString('base64');

    // Validate against ntfy directly
    ngx.fetch('http://127.0.0.1:8080/v1/account', {
        headers: { 'Authorization': 'Basic ' + encoded }
    }).then(function(reply) {
        if (reply.status >= 200 && reply.status < 300) {
            // Valid credentials — set cookie and redirect to /
            var cookieValue = 'ntfy_auth=' + encoded +
                '; Path=/' +
                '; HttpOnly' +
                '; SameSite=Strict' +
                '; Max-Age=2592000';
            r.headersOut['Set-Cookie'] = cookieValue;
            r.headersOut['Location'] = baseUrl(r) + '/';
            r.status = 302;
            r.sendHeader();
            r.finish();
        } else {
            r.headersOut['Location'] = baseUrl(r) + '/login?error=1';
            r.status = 302;
            r.sendHeader();
            r.finish();
        }
    }).catch(function(e) {
        r.headersOut['Location'] = '/login?error=1';
        r.status = 302;
        r.sendHeader();
        r.finish();
    });
}

function logout(r) {
    r.headersOut['Set-Cookie'] = 'ntfy_auth=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0';
    r.headersOut['Location'] = baseUrl(r) + '/login';
    r.status = 302;
    r.sendHeader();
    r.finish();
}

export default { getAuthHeader, checkCookie, loginPost, logout };
