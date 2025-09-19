import { Inject, Injectable, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Product } from '../domain/product';
import { Repository } from 'typeorm';
import { CACHE_MANAGER } from '@nestjs/cache-manager';
import { Cache } from 'cache-manager';
import { ClientProxy } from '@nestjs/microservices';

@Injectable()
export class ProductService {
  constructor(
    @InjectRepository(Product)
    private readonly productRepo: Repository<Product>,
    @Inject(CACHE_MANAGER) private cacheManager: Cache,
    @Inject('PRODUCT_PUBLISHER') private readonly client: ClientProxy,
  ) {}

  async findOne(id: number): Promise<Product> {
    const cacheKey = `product:${id}`;
    const cached = await this.cacheManager.get<Product>(cacheKey);
    if (cached) {
      return cached;
    }

    const product = await this.productRepo.findOne({ where: { id } });
    if (!product) throw new NotFoundException(`Product ${id} not found`);
    await this.cacheManager.set(cacheKey, product, 60_000);
    return product;
  }

  async create(data: Partial<Product>): Promise<Product> {
    const product = this.productRepo.create(data);
    const saved = await this.productRepo.save(product);

    this.client.emit('product_created', saved);
    return saved;
  }

  async decrementQty(productId: number): Promise<Product | null> {
    const product = await this.productRepo.findOne({
      where: { id: productId },
    });
    if (!product) return null;
    if (product.qty <= 0) return null;

    product.qty -= 1;
    return this.productRepo.save(product);
  }
}
