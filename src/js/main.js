import Alpine from "alpinejs";
import persist from "@alpinejs/persist";
import morph from "@alpinejs/morph";
import htmx from "htmx.org";

window.htmx = htmx;

Alpine.plugin(persist);
Alpine.plugin(morph);

window.Alpine = Alpine;
Alpine.start();

function Hi(name = "Alpine.js") {
    return {
        message: `Hello, ${name}!`,
        init() {
            console.log(this.message);
        },
    };
}

const InjectionPatterns = [
    // XSS (Cross-Site Scripting) patterns
    /<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi,
    /<iframe\b[^>]*>/gi,
    /<object\b[^>]*>/gi,
    /<embed\b[^>]*>/gi,
    /<link\b[^>]*>/gi,
    /<meta\b[^>]*>/gi,
    /javascript:/gi,
    /vbscript:/gi,
    /on\w+\s*=/gi, // event handlers like onclick, onload, etc.

    // SQL Injection patterns
    /(\b(select|union|insert|update|delete|drop|alter|create|exec|execute)\b)/gi,
    /(--|\#|\/\*|\*\/)/g, // SQL comments
    /(\bor\b|\band\b)\s+\d+\s*=\s*\d+/gi, // or 1=1, and 1=1
    /'\s*(or|and)\s*'.*'=/gi,

    // Command Injection patterns
    /(\||&|;|\$\(|\`)/g, // shell metacharacters
    /(nc|netcat|wget|curl|ping|nslookup|dig)\s/gi,
    /(rm|ls|cat|chmod|chown|ps|kill)\s/gi,

    // Path Traversal patterns
    /\.\.\//g, // directory traversal
    /\.\.\\/g, // Windows directory traversal

    // LDAP Injection patterns
    /(\*|\(|\)|\\|\||&)/g,

    // NoSQL Injection patterns
    /\$where/gi,
    /\$ne/gi,
    /\$gt/gi,
    /\$lt/gi,

    // Server-Side Template Injection patterns
    /\{\{.*\}\}/g, // template expressions
    /\$\{.*\}/g, // template literals
    /<\%.*\%>/g, // JSP/ASP templates

    // XML/XXE patterns
    /<!DOCTYPE/gi,
    /<!ENTITY/gi,
    /<\?xml/gi,

    // Additional suspicious patterns
    /(eval|exec|system|shell_exec|passthru|file_get_contents)/gi,
    /data:text\/html/gi, // data URLs
    /base64/gi, // base64 encoded content (potential)

    // Role manipulation attempts
    /ignore\s+(previous|all|above|prior)\s+(instructions|prompts|rules)/gi,
    /forget\s+(everything|all|previous|above)/gi,
    /you\s+are\s+(now|a|an)\s+(assistant|ai|bot|system|admin|developer)/gi,
    /act\s+as\s+(if\s+you\s+are\s+)?(a|an|the)?\s*(developer|admin|system|hacker|jailbreak)/gi,
    /pretend\s+(to\s+be|you\s+are)/gi,
    /roleplay\s+as/gi,

    // System prompt manipulation
    /system\s*[:=]\s*/gi,
    /assistant\s*[:=]\s*/gi,
    /user\s*[:=]\s*/gi,
    /\[system\]/gi,
    /\[assistant\]/gi,
    /\[user\]/gi,
    /<\|system\|>/gi,
    /<\|assistant\|>/gi,
    /<\|user\|>/gi,

    // Instruction override attempts
    /new\s+(instructions|rules|guidelines)/gi,
    /override\s+(previous|all|default)/gi,
    /disregard\s+(previous|all|above)/gi,
    /instead\s+of\s+.*(do|say|respond|answer)/gi,
    /stop\s+(following|using)\s+(instructions|rules)/gi,

    // Jailbreak attempts
    /jailbreak/gi,
    /dan\s+(mode|prompt)/gi, // "Do Anything Now"
    /evil\s+(mode|assistant|ai)/gi,
    /unrestricted\s+(mode|ai|assistant)/gi,
    /developer\s+mode/gi,
    /god\s+mode/gi,
    /admin\s+mode/gi,

    // Prompt injection markers
    /---\s*(ignore|stop|end)/gi,
    /```\s*(ignore|stop|end)/gi,
    /\[end\s+of\s+(prompt|instructions)\]/gi,
    /\[new\s+(prompt|instructions)\]/gi,

    // Token manipulation
    /special\s+token/gi,
    /end\s+of\s+context/gi,
    /context\s+window/gi,
    /token\s+limit/gi,

    // Prompt leakage attempts
    /show\s+(me\s+)?(your|the)\s+(prompt|instruction|system\s+message)/gi,
    /what\s+(are\s+)?(your|the)\s+(instruction|rules|guidelines)/gi,
    /repeat\s+(your|the)\s+(prompt|instruction)/gi,
    /output\s+(your|the)\s+(prompt|system\s+message)/gi,

    // Direct manipulation attempts
    /\+\+\+\s*(ignore|override|new)/gi,
    /!!!\s*(ignore|override|new)/gi,
    /###\s*(ignore|override|new)/gi,

    // Social engineering patterns
    /this\s+is\s+(urgent|important|critical)/gi,
    /emergency\s+(override|mode)/gi,
    /authorized\s+(by|request)/gi,
    /debugging\s+(mode|purpose)/gi,
    /testing\s+(mode|purpose)/gi,

    // Format manipulation
    /\<prompt\>/gi,
    /\<\/prompt\>/gi,
    /\<instruction\>/gi,
    /\<\/instruction\>/gi,

    // Language switching for evasion
    /translate\s+to\s+\w+\s*:/gi,
    /in\s+\w+\s+language/gi,
    /respond\s+in\s+\w+/gi,

    // Meta-instructions
    /simulate\s+(a|an|the)/gi,
    /hypothetically/gi,
    /in\s+a\s+fictional\s+scenario/gi,
    /for\s+(educational|research)\s+purposes/gi,

    // Code injection within prompts
    /```\s*python/gi,
    /```\s*javascript/gi,
    /```\s*bash/gi,
    /exec\s*\(/gi,
    /eval\s*\(/gi,

    // Markdown/formatting manipulation
    /\*\*\*\s*(ignore|override)/gi,
    /___\s*(ignore|override)/gi,

    // Chain-of-thought manipulation
    /step\s+by\s+step.*ignore/gi,
    /first.*then\s+ignore/gi,
    /think\s+step\s+by\s+step.*but\s+ignore/gi,
];

window.InjectionPatterns = InjectionPatterns;
window.Hi = Hi;