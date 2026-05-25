const UUID_EPOCH_OFFSET = 122192928000000000n

export function createUUIDv6Generator(
  now: () => number = Date.now,
  random: () => number = Math.random
): () => string {
  let lastTimestamp = 0n
  const clockSeq = BigInt((random() * 0x3fff) | 0)
  const node1 = BigInt((random() * 0xffff) | 0)
  const node2 = BigInt((random() * 0xffff) | 0)
  const node3 = BigInt((random() * 0xffff) | 0)

  return () => {
    let timestamp = BigInt(now()) * 10000n + UUID_EPOCH_OFFSET
    if (timestamp <= lastTimestamp) {
      timestamp = lastTimestamp + 1n
    }
    lastTimestamp = timestamp & 0x0fffffffffffffffn

    const timeHigh = (lastTimestamp >> 12n) & 0xffffffffffffn
    const timeLow = lastTimestamp & 0xfffn
    const h = (n: bigint, len: number) => n.toString(16).padStart(len, '0')

    return [
      h(timeHigh >> 16n, 8),
      h(timeHigh & 0xffffn, 4),
      h(0x6000n | timeLow, 4),
      h(0x8000n | clockSeq, 4),
      h((node1 << 32n) | (node2 << 16n) | node3, 12),
    ].join('-')
  }
}

const nextUUIDv6 = createUUIDv6Generator()

export function newUUIDv6(): string {
  return nextUUIDv6()
}
