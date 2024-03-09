export class Analytics {
    constructor(writeKey: string, endpoint: string, options?: Options);

    alias(
        userId: string,
        previousId?: string,
        options?: Record<string, unknown>,
        callback?: () => void
    ): Promise<SentEvent>;

    anonymize(): Promise<SentEvent>;

    close(): void;

    debug(on: boolean): void;

    endSession(): void;

    getAnonymousId(): string;

    getSessionId(): number;

    group(): Group;
    group(
        groupId: string,
        traits?: Record<string, unknown>,
        options?: Record<string, unknown>,
        callback?: () => void
    ): Promise<SentEvent>;
    group(
        traits: Record<string, unknown>,
        options?: Record<string, unknown>,
        callback?: () => void
    ): Promise<SentEvent>;

    identify(
        userId: string,
        traits?: Record<string, unknown>,
        options?: Record<string, unknown>,
        callback?: () => void
    ): Promise<SentEvent>;
    identify(
        traits?: string,
        options?: Record<string, unknown>,
        callback?: () => void
    ): Promise<SentEvent>;

    page(
        name: string,
        properties?: Record<string, unknown>,
        options?: Record<string, unknown>,
        callback?: () => void
    ): Promise<SentEvent>;
    page(
        category: string,
        name: string,
        properties?: Record<string, unknown>,
        options?: Record<string, unknown>,
        callback?: () => void
    ): Promise<SentEvent>;
    page(
        properties?: Record<string, unknown>,
        options?: Record<string, unknown>,
        callback?: () => void
    ): Promise<SentEvent>;

    ready(callback?: () => void): Promise<void>;

    reset(): void;

    screen(
        name: string,
        properties?: Record<string, unknown>,
        options?: Record<string, unknown>,
        callback?: () => void
    ): Promise<SentEvent>;
    screen(
        category: string,
        name: string,
        properties?: Record<string, unknown>,
        options?: Record<string, unknown>,
        callback?: () => void
    ): Promise<SentEvent>;
    screen(
        properties?: Record<string, unknown>,
        options?: Record<string, unknown>,
        callback?: () => void
    ): Promise<SentEvent>;

    setAnonymousId(id?: string): string;

    startSession(id?: number): void;

    track(
        name: string,
        properties?: Record<string, unknown>,
        options?: Record<string, unknown>,
        callback?: () => void
    ): Promise<SentEvent>;

    user(): User;
}

export enum SameSite {
    Lax = "Lax",
    Strict = "Strict",
    None = "None",
}

export enum StorageType {
    multiStorage = "multiStorage",
    cookieStorage = "cookieStorage",
    localStorage = "localStorage",
    sessionStorage = "sessionStorage",
    none = "none",
}

interface UseQueryString {
    aid: RegExp;
    uid: RegExp;
}

interface Options {
    debug?: boolean;
    sessions?: {
        autoTrack?: boolean;
        timeout?: number;
    };
    storage?: {
        cookie?: {
            domain?: string | null;
            maxAge?: number;
            path?: string;
            sameSite?: SameSite;
            secure?: boolean;
        };
        type?: StorageType;
    };
    useQueryString?: boolean | UseQueryString;
}

interface SentEvent {
    attempts: number;
    event: Event;
}

interface Event {
    anonymousId: string;
    category?: string;
    context: {
        campaign?: Record<string, string>;
        library: {
            name: string;
            version: string;
        };
        locale: string;
        page: {
            path: string;
            referrer: string;
            search: string;
            title: string;
            url: string;
        };
        screen: {
            width: number;
            height: number;
            density: number;
        };
        sessionId?: number;
        sessionStart?: boolean;
        userAgent: string;
        [property: string]: unknown;
    };
    event?: string;
    groupId?: string;
    integrations: Record<string, unknown>;
    messageId: string;
    name?: string;
    properties?: Record<string, unknown>;
    previousId?: string;
    timestamp: Date;
    traits?: Record<string, unknown>;
    type: string;
    userId: string | null;
}

export class Group {
    id(id?: string | null): string | null;
    traits(traits?: Record<string, unknown> | null): Record<string, unknown>;
}

export class User {
    anonymousId(id?: string | null): string;
    id(id?: string | null): string | null;
    traits(traits?: Record<string, unknown> | null): Record<string, unknown>;
}
