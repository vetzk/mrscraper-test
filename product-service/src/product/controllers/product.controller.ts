import {
  Body,
  Controller,
  Param,
  ParseIntPipe,
  Post,
  Get,
  Put,
  Logger,
  Inject,
} from '@nestjs/common';
import { ProductService } from '../services/product.service';
import { Product } from '../domain/product';
import { CreateProductDto } from '../dtos/create-product.dto';
import {
  ClientProxy,
  Ctx,
  EventPattern,
  Payload,
  RmqContext,
} from '@nestjs/microservices';

@Controller('products')
export class ProductController {
  private readonly logger = new Logger(ProductService.name);

  constructor(
    private readonly productService: ProductService,
    @Inject('PRODUCT_PUBLISHER') private readonly client: ClientProxy,
  ) {}

  @Get(':id')
  findOne(@Param('id', ParseIntPipe) id: number): Promise<Product> {
    return this.productService.findOne(id);
  }

  @Post()
  create(@Body() dto: CreateProductDto): Promise<Product> {
    return this.productService.create(dto);
  }

  @EventPattern('order.created')
  async handleOrderCreated(@Payload() data: any) {
    const { orderId, productId } = data;
    this.logger.log(
      `Received order.created for order ${orderId}, product ${productId}`,
    );

    const product = await this.productService.decrementQty(productId);

    if (!product) {
      this.logger.log(`Order failed. Insufficient stock`);
      await this.client
        .emit('order.qty_failed', {
          orderId,
          reason: 'product_not_found_or_unavailable',
        })
        .toPromise();
      return;
    }

    await this.client.emit('order.qty_confirmed', { orderId }).toPromise();
  }

  @EventPattern('*')
  async handleAny(@Payload() data: any, @Ctx() ctx: RmqContext) {
    this.logger.warn(`⚠️ Wildcard caught pattern: ${ctx.getPattern()}`);
    this.logger.warn(`Raw data: ${JSON.stringify(data)}`);
  }
}
